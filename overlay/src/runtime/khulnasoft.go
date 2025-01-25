package runtime

import (
	"unsafe"
)

// setKhulnsoftG sets the khulnsoftG value on the running g to the given value.
func setKhulnsoftG(val unsafe.Pointer) {
	g := getg().m.curg
	g.khulnsoft = val
}

// getKhulnsoftG gets the khulnsoftG value from the running g.
func getKhulnsoftG() unsafe.Pointer {
	return getg().m.curg.khulnsoft
}

// khulnsoftCallers is like runtime.Callers but also returns the offset
// of the text segment to make the PCs ASLR-independent.
func khulnsoftCallers(skip int, pc []uintptr) (n int, off uintptr) {
	n = Callers(skip+1, pc)
	return n, firstmoduledata.text
}

// To allow Khulnsoft to use go:linkname:

//go:linkname getKhulnsoftG
//go:linkname setKhulnsoftG
//go:linkname khulnsoftCallers
