package driver

import (
	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"sync"
)

// NonBlockingGRPCServer defines Non blocking GRPC server interfaces
type NonBlockingGRPCServer interface {
	// Start services at the endpoint
	Start(endpoint string)
	// Wait wWaits for the service to stop
	Wait()
	// Stop stops the service gracefully
	Stop()
	// ForceStop stops the service forcefully
	ForceStop()
}

type Driver struct {
	identityServer   csi.IdentityServer
	controllerServer csi.ControllerServer
	nodeServer       csi.NodeServer

	log    *logrus.Entry
	wg     sync.WaitGroup
	server *grpc.Server
}
