package mounter

import (
	"github.com/sirupsen/logrus"
	"k8s.io/mount-utils"
	"os"
	"path/filepath"
)

const (
	runningState = "running"
)

const (
	// blkidExitStatusNoIdentifiers defines the exit code returned from blkid indicating that no devices have been found. See http://www.polarhome.com/service/man/?qf=blkid&tf=2&of=Alpinelinux for details.
	blkidExitStatusNoIdentifiers = 2
)

// Interface is responsible for formatting and mounting volumes
type Interface interface {
	// Format formats the source with the given filesystem type
	Format(source, fsType string) error

	// Mount mounts source to target with the given fstype and options.
	Mount(source, target, fsType string, options ...string) error

	// Unmount unmounts the given target
	Unmount(target string) error

	// IsAttached checks whether the source device is in the running state.
	IsAttached(source string) error

	// IsFormatted checks whether the source device is formatted or not. It
	// returns true if the source device is already formatted.
	IsFormatted(source string) (bool, error)

	// IsMounted checks whether the target path is a correct mount (i.e:
	// propagated). It returns true if it's mounted. An error is returned in
	// case of system errors or if it's mounted incorrectly.
	IsMounted(target string) (bool, error)

	DevicePathByVolumeID(volumeID string) (string, error)

	// GetStatistics returns capacity-related volume statistics for the given
	// volume path.
	GetStatistics(volumePath string) (*VolumeStatistics, error)

	// IsBlockDevice checks whether the device at the path is a block device
	IsBlockDevice(volumePath string) (bool, error)
}
type Mounter struct {
	log                 *logrus.Entry
	kMounter            *mount.SafeFormatAndMount
	attachmentValidator AttachmentValidator
}

type AttachmentValidator interface {
	readFile(name string) ([]byte, error)
	evalSymlinks(path string) (string, error)
}
type prodAttachmentValidator struct{}

func (av *prodAttachmentValidator) readFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}
func (av *prodAttachmentValidator) evalSymlinks(path string) (string, error) {
	return filepath.EvalSymlinks(path)
}

type VolumeStatistics struct {
	AvailableBytes, TotalBytes, UsedBytes    int64
	AvailableInodes, TotalInodes, UsedInodes int64
}

type findmntResponse struct {
	FileSystems []fileSystem `json:"filesystems"`
}
type fileSystem struct {
	Target      string `json:"target"`
	Propagation string `json:"propagation"`
	FsType      string `json:"fstype"`
	Options     string `json:"options"`
}
