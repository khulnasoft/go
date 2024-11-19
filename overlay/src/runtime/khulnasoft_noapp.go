//go:build !khulnasoft
// +build !khulnasoft

package runtime

import "unsafe"

// startKhulnasoftG starts a new g, copying the src khulnasoft data to a new khulnasoftG value.
// It is defined here as a no-op for Khulnasoft binaries that were built
// without linking in the Khulnasoft runtime to avoid strange build errors.
func startKhulnasoftG(src unsafe.Pointer) unsafe.Pointer {
	return nil
}

// exitKhulnasoftG marks a goroutine as having exited.
// It is defined here as a no-op for Khulnasoft binaries that were built
// without linking in the Khulnasoft runtime to avoid strange build errors.
func exitKhulnasoftG(src unsafe.Pointer) {
}
