package controller

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	"github.com/Edge-Center/edgecentercloud-go/v2/util"
	"github.com/sirupsen/logrus"
)

func (s *Service) ensureVolumeFromNew(ctx context.Context, name, vtype string, size int, md edgecloudV2.Metadata) (*edgecloudV2.Volume, error) {
	opt := &edgecloudV2.VolumeCreateRequest{
		Source:   edgecloudV2.VolumeSourceNewVolume,
		Size:     size,
		TypeName: volumeType(vtype),
		Name:     name,
		Metadata: md,
	}
	result, err := util.ExecuteAndExtractTaskResult(ctx, s.cloud.Volumes.Create, opt, s.cloud, 2*time.Minute)
	if err != nil {
		return nil, err
	}
	vol, _, err := s.cloud.Volumes.Get(ctx, result.Volumes[0])
	return vol, err
}
func (s *Service) ensureVolumeFromSnapshot(ctx context.Context, name, snapshotID, vtype string, size int, md edgecloudV2.Metadata) (*edgecloudV2.Volume, error) {
	opt := &edgecloudV2.VolumeCreateRequest{
		Source:     edgecloudV2.VolumeSourceSnapshot,
		Size:       size,
		TypeName:   volumeType(vtype),
		Name:       name,
		Metadata:   md,
		SnapshotID: snapshotID,
	}
	result, err := util.ExecuteAndExtractTaskResult(ctx, s.cloud.Volumes.Create, opt, s.cloud, 2*time.Minute)
	if err != nil {
		return nil, err
	}
	vol, _, err := s.cloud.Volumes.Get(ctx, result.Volumes[0])
	if err != nil {
		return nil, err
	}
	vol.SnapshotIDs = append(vol.SnapshotIDs, snapshotID)
	return vol, err
}

func (s *Service) ensureAttachmentVolume(ctx context.Context, volumeID, instanceID string) (string, error) {
	exist, err := util.ResourceIsExist(ctx, s.cloud.Volumes.Get, volumeID)
	if err != nil {
		return "", err
	}
	if !exist {
		return "", errors.New("not found")
	}

	exist, err = util.ResourceIsExist(ctx, s.cloud.Instances.Get, instanceID)
	if err != nil {
		return "", err
	}
	if !exist {
		return "", errors.New("not found")
	}

	vol, _, err := s.cloud.Volumes.Get(ctx, volumeID)
	if err != nil {
		return "", err
	}
	if len(vol.Attachments) != 0 {
		if vol.Attachments[0].ServerID == instanceID {
			return "", nil
		}
		return "", errors.New("volume attach to different node")
	}

	_, _, err = s.cloud.Volumes.Attach(ctx, volumeID, &edgecloudV2.VolumeAttachRequest{InstanceID: instanceID})
	if err != nil {
		return "", err
	}
	if err = util.WaitVolumeAttachedToInstance(ctx, s.cloud, volumeID, instanceID, nil); err != nil {
		return "", err
	}

	vol, _, err = s.cloud.Volumes.Get(ctx, volumeID)
	if err != nil {
		return "", err
	}
	if vol.Status != "in-use" {
		return "", fmt.Errorf("cannot get device path of volume %s, its status is %s ", vol.Name, vol.Status)
	}

	var devicePath string
	if len(vol.Attachments) > 0 && vol.Attachments[0].ServerID != "" {
		if instanceID == vol.Attachments[0].ServerID {
			devicePath = vol.Attachments[0].Device
		}
		return "", fmt.Errorf("[ControllerPublishVolume] disk %q is attached to a different compute: %q, should be detached before proceeding",
			vol.ID, vol.Attachments[0].ServerID)
	}
	return devicePath, nil
}
func (s *Service) ensureDetachmentVolume(ctx context.Context, volumeID, instanceID string) error {
	exist, err := util.ResourceIsExist(ctx, s.cloud.Instances.Get, instanceID)
	if err != nil {
		return err
	}
	if !exist {
		return errors.New("not found")
	}

	_, _, err = s.cloud.Volumes.Detach(ctx, volumeID, &edgecloudV2.VolumeDetachRequest{InstanceID: instanceID})
	if err != nil {
		return err
	}
	return util.WaitVolumeDetachedFromInstance(ctx, s.cloud, volumeID, instanceID, nil)
}

func (s *Service) ensureExpandingVolume(ctx context.Context, volumeID string, size int) error {
	volume, resp, err := s.cloud.Volumes.Get(ctx, volumeID)
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusNotFound {
		return errors.New("volume not found")
	}

	if volume.Size >= size {
		// a volume was already resized
		s.log.WithFields(logrus.Fields{"current_volume_size": volume.Size, "requested_volume_size": size}).Info("skipping volume resize because current volume size exceeds requested volume size")
		// even if the volume is resized independently of the control panel, we still need to resize the node fs when resize is requested
		// in this case, the claim capacity will be resized to the volume capacity, requested capacity will be ignored to make the PV and PVC capacities consistent
		return nil
	}

	task, _, err := s.cloud.Volumes.Extend(ctx, volumeID, &edgecloudV2.VolumeExtendSizeRequest{Size: size})
	if err != nil {
		return err
	}

	if err = util.WaitForTaskComplete(ctx, s.cloud, task.Tasks[0]); err != nil {
		return err
	}
	return nil
}

func (s *Service) ensureSnapshot(ctx context.Context, volumeID, name string, md edgecloudV2.Metadata) (*edgecloudV2.Snapshot, error) {
	opt := &edgecloudV2.SnapshotCreateRequest{
		VolumeID: volumeID,
		Name:     name,
		Metadata: md,
	}

	result, err := util.ExecuteAndExtractTaskResult(ctx, s.cloud.Snapshots.Create, opt, s.cloud, 2*time.Minute)

	if err != nil {
		return nil, err
	}

	snapshot, _, err := s.cloud.Snapshots.Get(ctx, result.Snapshots[0])

	return snapshot, err
}
