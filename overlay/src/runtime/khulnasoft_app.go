//go:build khulnsoft
// +build khulnsoft

package runtime

import "unsafe"

// startKhulnsoftG starts a new g, copying the src khulnsoft data to a new khulnsoftG value.
// It must be defined by the Khulnsoft runtime and linked using
// go:linkname.
func startKhulnsoftG(src unsafe.Pointer) unsafe.Pointer

// exitKhulnsoftG marks a goroutine as having exited.
// It must be defined by the Khulnsoft runtime and linked using
// go:linkname.
func exitKhulnsoftG(src unsafe.Pointer)
