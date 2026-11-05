package node

import (
	"ec-csi-plugin/pkg/metadata"
	"ec-csi-plugin/pkg/mounter"
	"ec-csi-plugin/pkg/region"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/sirupsen/logrus"
)

type Service struct {
	mounter  mounter.Interface
	metadata metadata.Interface
	regions  *region.Set

	log *logrus.Entry

	validateAttachment bool
	csi.UnimplementedNodeServer
}
