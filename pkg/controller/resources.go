package controller

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	"github.com/Edge-Center/edgecentercloud-go/v2/util"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const defaultTaskTimeout = 2 * time.Minute

func (s *Service) ensureVolumeFromNew(ctx context.Context, name, vtype string, size int, md edgecloudV2.Metadata) (*edgecloudV2.Volume, error) {
	opt := &edgecloudV2.VolumeCreateRequest{
		Source:   edgecloudV2.VolumeSourceNewVolume,
		Size:     size,
		TypeName: volumeType(vtype),
		Name:     name,
		Metadata: md,
	}
	result, err := util.ExecuteAndExtractTaskResult(ctx, s.cloud.Volumes.Create, opt, s.cloud, defaultTaskTimeout)
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
	result, err := util.ExecuteAndExtractTaskResult(ctx, s.cloud.Volumes.Create, opt, s.cloud, defaultTaskTimeout)
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

// findAttachment returns the attachment record of the given instance, if the volume is attached to it.
func findAttachment(attachments []edgecloudV2.Attachment, instanceID string) (edgecloudV2.Attachment, bool) {
	for _, attachment := range attachments {
		if attachment.ServerID == instanceID {
			return attachment, true
		}
	}
	return edgecloudV2.Attachment{}, false
}

// foreignServerIDs returns the instances the volume is attached to, except the given one.
func foreignServerIDs(attachments []edgecloudV2.Attachment, instanceID string) []string {
	ids := make([]string, 0, len(attachments))
	for _, attachment := range attachments {
		if attachment.ServerID != instanceID {
			ids = append(ids, attachment.ServerID)
		}
	}
	return ids
}

func (s *Service) ensureAttachmentVolume(ctx context.Context, volumeID, instanceID string) (string, error) {
	exist, err := util.ResourceIsExist(ctx, s.cloud.Volumes.Get, volumeID)
	if err != nil {
		return "", err
	}

	if !exist {
		return "", status.Errorf(codes.NotFound, "volume %q not found", volumeID)
	}

	exist, err = util.ResourceIsExist(ctx, s.cloud.Instances.Get, instanceID)
	if err != nil {
		return "", err
	}

	if !exist {
		return "", status.Errorf(codes.NotFound, "instance %q not found", instanceID)
	}

	vol, _, err := s.cloud.Volumes.Get(ctx, volumeID)
	if err != nil {
		return "", err
	}

	s.logExtraAttachments(vol, instanceID, "before attach")

	// the volume is already attached to the requested instance, nothing to do
	if attachment, ok := findAttachment(vol.Attachments, instanceID); ok {
		s.log.WithFields(
			logrus.Fields{"volume_id": volumeID, "instance_id": instanceID},
		).Info("volume is already attached to the instance")
		return attachment.Device, nil
	}

	// only ReadWriteOnce is supported, so an attachment to another instance is a real conflict
	if foreign := foreignServerIDs(vol.Attachments, instanceID); len(foreign) > 0 {
		return "", status.Errorf(codes.FailedPrecondition,
			"volume %q is attached to a different compute: %q, it should be detached before proceeding",
			volumeID,
			strings.Join(foreign, ", "),
		)
	}

	_, _, err = s.cloud.Volumes.Attach(ctx, volumeID, &edgecloudV2.VolumeAttachRequest{
		InstanceID: instanceID,
	})
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

	s.logExtraAttachments(vol, instanceID, "after attach")

	if vol.Status != "in-use" {
		return "", fmt.Errorf("cannot get device path of volume %s, its status is %s",
			vol.Name,
			vol.Status,
		)
	}

	attachment, ok := findAttachment(vol.Attachments, instanceID)
	if !ok {
		return "", fmt.Errorf("volume %q is reported as attached to %q, but no attachment info was returned",
			volumeID, instanceID)
	}

	return attachment.Device, nil
}

// logExtraAttachments reports the attachment records of a volume when more than the expected one is present.
func (s *Service) logExtraAttachments(vol *edgecloudV2.Volume, instanceID, stage string) {
	if len(foreignServerIDs(vol.Attachments, instanceID)) == 0 {
		return
	}
	records := make([]string, 0, len(vol.Attachments))
	for _, attachment := range vol.Attachments {
		records = append(
			records,
			fmt.Sprintf("{server_id: %s, volume_id: %s, attachment_id: %s, device: %s, attached_at: %s}",
				attachment.ServerID, attachment.VolumeID, attachment.AttachmentID, attachment.Device, attachment.AttachedAt,
			),
		)
	}
	s.log.WithFields(logrus.Fields{
		"volume_id":     vol.ID,
		"volume_status": vol.Status,
		"instance_id":   instanceID,
		"stage":         stage,
		"attachments":   strings.Join(records, ", "),
	}).Warn("volume has attachment records of foreign instances")
}

func (s *Service) ensureDetachmentVolume(ctx context.Context, volumeID, instanceID string) error {
	vol, resp, err := s.cloud.Volumes.Get(ctx, volumeID)
	if resp != nil && resp.StatusCode == http.StatusNotFound {
		s.log.WithField("volume_id", volumeID).Info("volume does not exist, skipping detach")
		return nil
	}
	if err != nil {
		return err
	}
	if len(vol.Attachments) == 0 {
		s.log.WithField("volume_id", volumeID).Info("volume is not attached, skipping detach")
		return nil
	}
	attachedToInstance := slices.ContainsFunc(vol.Attachments, func(a edgecloudV2.Attachment) bool {
		return a.ServerID == instanceID
	})
	if !attachedToInstance {
		s.log.WithField("volume_id", volumeID).Info("volume is not attached to this instance, skipping detach")
		return nil
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

	if err = util.WaitForTaskComplete(ctx, s.cloud, task.Tasks[0], defaultTaskTimeout); err != nil {
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

	result, err := util.ExecuteAndExtractTaskResult(ctx, s.cloud.Snapshots.Create, opt, s.cloud, defaultTaskTimeout)

	if err != nil {
		return nil, err
	}

	snapshot, _, err := s.cloud.Snapshots.Get(ctx, result.Snapshots[0])

	return snapshot, err
}
