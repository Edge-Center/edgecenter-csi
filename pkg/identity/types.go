package identity

import "github.com/container-storage-interface/spec/lib/go/csi"

type Service struct {
	Name    string
	Version string
	csi.UnimplementedIdentityServer
}
