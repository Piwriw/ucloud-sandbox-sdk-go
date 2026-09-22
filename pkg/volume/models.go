// Package volume manages persistent volumes.
//
// A volume outlives the sandboxes that mount it, so it is where anything worth
// keeping goes. Create one, then mount it wherever it is needed:
//
//	v, err := c.Volumes().Create(ctx, "datasets")
//
//	sbx, err := c.Sandboxes().Create(ctx, sandbox.CreateOptions{
//	    VolumeMounts: []api.SandboxVolumeMount{v.Mount("/mnt/data")},
//	})
//
// Reading and writing a volume's contents happens from inside a sandbox that
// has it mounted. There is a separate content API in the upstream E2B
// platform, reached with the per-volume token Service.Get returns, but UCloud
// does not serve it, so this package does not wrap it.
package volume

import "github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"

// Info is a volume's identity.
type Info struct {
	VolumeID string
	Name     string
}

// Volume is a handle to one volume. Get one from Service.Create or
// Service.Connect.
type Volume struct {
	ID   string
	Name string

	svc *Service
}

// Mount describes mounting this volume at path inside a sandbox, in the shape
// sandbox.CreateOptions expects.
//
// The generated type is used directly so that pkg/sandbox does not have to
// import this package to accept a mount.
func (v *Volume) Mount(path string) api.SandboxVolumeMount {
	return api.SandboxVolumeMount{Name: v.Name, Path: path}
}

// infoFrom converts a generated Volume into an Info.
func infoFrom(v api.Volume) Info {
	return Info{VolumeID: v.VolumeID, Name: v.Name}
}
