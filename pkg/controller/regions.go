package controller

import (
	"context"
	"errors"

	"ec-csi-plugin/pkg/region"

	edgecloud "github.com/Edge-Center/edgecentercloud-go/v2"
	"github.com/Edge-Center/edgecentercloud-go/v2/util"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Service) inScope(scope region.Scope) *Service {
	return &Service{cloud: scope.Client, regions: s.regions, scope: scope, log: s.log, clusterID: s.clusterID}
}

func (s *Service) forHandle(handle string) (*Service, string, error) {
	scope, id, err := s.regions.ForHandle(handle)
	if err != nil {
		return nil, "", err
	}
	return s.inScope(scope), id, nil
}

func (s *Service) findVolume(ctx context.Context, name string) ([]edgecloud.Volume, error) {
	var result []edgecloud.Volume
	for _, scope := range s.regions.Scopes {
		volumes, err := util.VolumesListByName(ctx, scope.Client, name)
		if err != nil && !errors.Is(err, util.ErrVolumesNotFound) {
			return nil, status.Errorf(codes.Unavailable, "lookup volume in CSI scope %q: %v", scope.Name, err)
		}
		if len(volumes) == 0 {
			continue
		}
		if scope.Config != s.scope.Config {
			return nil, status.Errorf(codes.FailedPrecondition, "volume %q already exists in CSI scope %q", name, scope.Name)
		}
		result = volumes
	}
	return result, nil
}

func (s *Service) createResponse(volume *edgecloud.Volume, req *csi.CreateVolumeRequest) *csi.CreateVolumeResponse {
	response := prepareCreateResponse(volume, true, req.GetAccessibilityRequirements())
	response.Volume.VolumeId = s.scope.Handle(volume.ID)
	if s.regions.Enabled() {
		response.Volume.AccessibleTopology = []*csi.Topology{s.scope.Topology()}
	}
	response.Volume.ContentSource = req.GetVolumeContentSource()
	return response
}
