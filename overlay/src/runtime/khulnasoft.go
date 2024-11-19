package runtime

import (
	"unsafe"
)

// setKhulnasoftG sets the khulnasoftG value on the running g to the given value.
func setKhulnasoftG(val unsafe.Pointer) {
	g := getg().m.curg
	g.khulnasoft = val
}

// getKhulnasoftG gets the khulnasoftG value from the running g.
func getKhulnasoftG() unsafe.Pointer {
	return getg().m.curg.khulnasoft
}

// khulnasoftCallers is like runtime.Callers but also returns the offset
// of the text segment to make the PCs ASLR-independent.
func khulnasoftCallers(skip int, pc []uintptr) (n int, off uintptr) {
	n = Callers(skip+1, pc)
	return n, firstmoduledata.text
}

// To allow Khulnasoft to use go:linkname:

//go:linkname getKhulnasoftG
//go:linkname setKhulnasoftG
//go:linkname khulnasoftCallers
