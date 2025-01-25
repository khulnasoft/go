//go:build !khulnsoft
// +build !khulnsoft

package runtime

import "unsafe"

// startKhulnsoftG starts a new g, copying the src khulnsoft data to a new khulnsoftG value.
// It is defined here as a no-op for Khulnsoft binaries that were built
// without linking in the Khulnsoft runtime to avoid strange build errors.
func startKhulnsoftG(src unsafe.Pointer) unsafe.Pointer {
	return nil
}

// exitKhulnsoftG marks a goroutine as having exited.
// It is defined here as a no-op for Khulnsoft binaries that were built
// without linking in the Khulnsoft runtime to avoid strange build errors.
func exitKhulnsoftG(src unsafe.Pointer) {
}
