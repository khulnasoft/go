//go:build khulnasoft
// +build khulnasoft

package runtime

import "unsafe"

// startKhulnasoftG starts a new g, copying the src khulnasoft data to a new khulnasoftG value.
// It must be defined by the Khulnasoft runtime and linked using
// go:linkname.
//go:linkname startKhulnasoftG khulnasoft.runtime.startKhulnasoftG
func startKhulnasoftG(src unsafe.Pointer) unsafe.Pointer

// exitKhulnasoftG marks a goroutine as having exited.
// It must be defined by the Khulnasoft runtime and linked using
// go:linkname.
//go:linkname exitKhulnasoftG khulnasoft.runtime.exitKhulnasoftG
func exitKhulnasoftG(src unsafe.Pointer)
