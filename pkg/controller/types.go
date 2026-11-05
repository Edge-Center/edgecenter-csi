package controller

import (
	"ec-csi-plugin/pkg/region"
	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/sirupsen/logrus"
	"sync"
)

type Service struct {
	cloud    *edgecloudV2.Client
	regions  *region.Set
	scope    region.Scope
	creating sync.Map

	log *logrus.Entry

	clusterID string
	csi.UnimplementedControllerServer
}
