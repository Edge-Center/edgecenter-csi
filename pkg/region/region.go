package region

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	edgecloud "github.com/Edge-Center/edgecentercloud-go/v2"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const TopologyKey = "topology.csi.edgecenter.org/region"

type Config struct {
	ProjectID int `mapstructure:"projectID"`
	RegionID  int `mapstructure:"regionID"`
}

type Scope struct {
	Name string
	Config
	Client *edgecloud.Client
}

type Set struct {
	Scopes []Scope
}

func New(cfg edgecloud.CloudConfig, extra map[string]Config) (*Set, error) {
	if len(extra) > 0 && (cfg.ProjectID <= 0 || cfg.RegionID <= 0) {
		return nil, fmt.Errorf("multi-region CSI requires positive primary projectID and regionID")
	}
	names := make([]string, 0, len(extra))
	for name := range extra {
		if name == "" || name == "default" {
			return nil, fmt.Errorf("invalid additional region name %q", name)
		}
		names = append(names, name)
	}
	sort.Strings(names)
	set := &Set{}
	seen := map[Config]bool{}
	for _, name := range append([]string{"default"}, names...) {
		scope := extra[name]
		if name == "default" {
			scope = Config{ProjectID: cfg.ProjectID, RegionID: cfg.RegionID}
		} else {
			if scope.ProjectID == 0 {
				scope.ProjectID = cfg.ProjectID
			}
			if scope.ProjectID <= 0 || scope.RegionID <= 0 {
				return nil, fmt.Errorf("invalid projectID or regionID for %q", name)
			}
		}
		if seen[scope] {
			return nil, fmt.Errorf("duplicate project and region for %q", name)
		}
		seen[scope] = true
		client, err := edgecloud.NewWithRetries(nil, edgecloud.SetAPIKey(cfg.APIToken), edgecloud.SetBaseURL(cfg.APIUrl),
			edgecloud.SetProject(scope.ProjectID), edgecloud.SetRegion(scope.RegionID))
		if err != nil {
			return nil, fmt.Errorf("configure CSI region %q: %w", name, err)
		}
		set.Scopes = append(set.Scopes, Scope{Name: name, Config: scope, Client: client})
	}
	return set, nil
}

func (s *Set) Enabled() bool { return s != nil && len(s.Scopes) > 1 }

func (s Scope) key() string { return fmt.Sprintf("%d-%d", s.ProjectID, s.RegionID) }

func (s Scope) Topology() *csi.Topology {
	return &csi.Topology{Segments: map[string]string{TopologyKey: s.key()}}
}

func (s Scope) Handle(id string) string {
	if id == "" || s.Name == "default" || s.Name == "" {
		return id
	}
	return fmt.Sprintf("ec:%d:%d:%s", s.ProjectID, s.RegionID, id)
}

func SplitHandle(handle string) (Config, string, error) {
	if !strings.HasPrefix(handle, "ec:") {
		return Config{}, handle, nil
	}
	parts := strings.Split(handle, ":")
	if len(parts) != 4 || parts[3] == "" {
		return Config{}, "", status.Error(codes.InvalidArgument, "malformed regional CSI handle")
	}
	project, pErr := strconv.Atoi(parts[1])
	region, rErr := strconv.Atoi(parts[2])
	if pErr != nil || rErr != nil || project <= 0 || region <= 0 {
		return Config{}, "", status.Error(codes.InvalidArgument, "invalid project or region in CSI handle")
	}
	return Config{ProjectID: project, RegionID: region}, parts[3], nil
}

func (s *Set) ForHandle(handle string) (Scope, string, error) {
	scope, id, err := SplitHandle(handle)
	if err != nil {
		return Scope{}, "", err
	}
	if scope == (Config{}) {
		return s.Scopes[0], id, nil
	}
	for _, candidate := range s.Scopes {
		if candidate.Config == scope {
			return candidate, id, nil
		}
	}
	return Scope{}, "", status.Error(codes.FailedPrecondition, "CSI handle references an unconfigured project or region")
}

func (s *Set) ForTopology(req *csi.TopologyRequirement) (Scope, error) {
	if !s.Enabled() || req == nil {
		return s.Scopes[0], nil
	}
	key := ""
	if preferred := req.GetPreferred(); len(preferred) > 0 {
		key = preferred[0].GetSegments()[TopologyKey]
	} else {
		for _, topology := range req.GetRequisite() {
			candidate := topology.GetSegments()[TopologyKey]
			if key != "" && key != candidate {
				return Scope{}, status.Error(codes.InvalidArgument, "multiple CSI regions require a preferred topology")
			}
			key = candidate
		}
	}
	if key == "" && len(req.GetPreferred()) == 0 && len(req.GetRequisite()) == 0 {
		return s.Scopes[0], nil
	}
	if len(req.GetRequisite()) > 0 {
		allowed := false
		for _, topology := range req.GetRequisite() {
			allowed = allowed || topology.GetSegments()[TopologyKey] == key
		}
		if !allowed {
			return Scope{}, status.Error(codes.InvalidArgument, "preferred CSI region is outside allowed topology")
		}
	}
	for _, scope := range s.Scopes {
		if key == scope.key() {
			return scope, nil
		}
	}
	return Scope{}, status.Error(codes.FailedPrecondition, "node topology references an unconfigured CSI region")
}

func (s *Set) ForNode(ctx context.Context, id string) (Scope, error) {
	var found Scope
	for _, scope := range s.Scopes {
		vm, resp, err := scope.Client.Instances.Get(ctx, id)
		if err != nil {
			if resp != nil && resp.Response != nil && resp.StatusCode == http.StatusNotFound {
				continue
			}
			return Scope{}, status.Errorf(codes.Unavailable, "cannot verify node in CSI scope %q: %v", scope.Name, err)
		}
		if vm == nil || vm.ID != id {
			return Scope{}, status.Errorf(codes.Internal, "invalid node lookup result in CSI scope %q", scope.Name)
		}
		if found.Name != "" {
			return Scope{}, status.Error(codes.FailedPrecondition, "node UUID exists in multiple CSI scopes")
		}
		found = scope
	}
	if found.Name == "" {
		return Scope{}, status.Error(codes.NotFound, "node UUID not found in configured CSI scopes")
	}
	return found, nil
}
