package identity

import "github.com/container-storage-interface/spec/lib/go/csi"

type Service struct {
	Topology bool
	Name     string
	Version  string
	csi.UnimplementedIdentityServer
}
