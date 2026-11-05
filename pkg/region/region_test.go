package region

import (
	"context"
	"errors"
	"net/http"
	"testing"

	edgecloud "github.com/Edge-Center/edgecentercloud-go/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type instanceLookup struct {
	edgecloud.InstancesService
	statusCode int
}

func (l instanceLookup) Get(_ context.Context, id string) (*edgecloud.Instance, *edgecloud.Response, error) {
	resp := &edgecloud.Response{Response: &http.Response{StatusCode: l.statusCode}}
	if l.statusCode == http.StatusOK {
		return &edgecloud.Instance{ID: id}, resp, nil
	}
	return nil, resp, errors.New("cloud API error")
}

func TestNodeRegionRequiresConfirmedScope(t *testing.T) {
	primary := Scope{Name: "default", Config: Config{ProjectID: 7, RegionID: 112},
		Client: &edgecloud.Client{Instances: instanceLookup{statusCode: http.StatusNotFound}}}
	dt := Scope{Name: "dt", Config: Config{ProjectID: 7, RegionID: 113},
		Client: &edgecloud.Client{Instances: instanceLookup{statusCode: http.StatusOK}}}
	set := &Set{Scopes: []Scope{primary, dt}}
	actual, err := set.ForNode(context.Background(), "vm-id")
	if err != nil || actual.Topology().GetSegments()[TopologyKey] != "7-113" {
		t.Fatalf("expected DT topology, got %v, %v", actual.Name, err)
	}
	primary.Client.Instances = instanceLookup{statusCode: http.StatusForbidden}
	if _, err := set.ForNode(context.Background(), "vm-id"); status.Code(err) != codes.Unavailable {
		t.Fatalf("403 must block topology discovery, got %v", err)
	}
}
