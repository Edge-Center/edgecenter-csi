package controller

import (
	"ec-csi-plugin/pkg/region"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/sirupsen/logrus"
)

func New(regions *region.Set, clusterID string, log *logrus.Entry) csi.ControllerServer {
	return &Service{cloud: regions.Scopes[0].Client, regions: regions, scope: regions.Scopes[0], clusterID: clusterID, log: log}
}
