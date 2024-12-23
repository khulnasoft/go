//go:build khulnasoft
// +build khulnasoft

package runtime

import "unsafe"

// startKhulnasoftG starts a new g, copying the src khulnasoft data to a new khulnasoftG value.
// It must be defined by the Khulnasoft runtime and linked using
// go:linkname.
func startKhulnasoftG(src unsafe.Pointer) unsafe.Pointer

// exitKhulnasoftG marks a goroutine as having exited.
// It must be defined by the Khulnasoft runtime and linked using
// go:linkname.
func exitKhulnasoftG(src unsafe.Pointer)
