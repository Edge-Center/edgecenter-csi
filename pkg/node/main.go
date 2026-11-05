package node

import (
	"ec-csi-plugin/pkg/metadata"
	"ec-csi-plugin/pkg/mounter"
	"ec-csi-plugin/pkg/region"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/sirupsen/logrus"
)

func New(mounter mounter.Interface, metadata metadata.Interface, validateAttachment bool, regions *region.Set, log *logrus.Entry) csi.NodeServer {
	return &Service{mounter: mounter, metadata: metadata, regions: regions, validateAttachment: validateAttachment, log: log}
}
