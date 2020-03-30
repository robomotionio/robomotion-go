package tl

import (
	"unsafe"
)

func Typelinks() (sections []unsafe.Pointer, offset [][]int32) {
	return typelinks()
}

func typelinks() (sections []unsafe.Pointer, offset [][]int32)

func Add(p unsafe.Pointer, x uintptr, whySafe string) unsafe.Pointer {
	return add(p, x, whySafe)
}

// add returns p+x.
//
// The whySafe string is ignored, so that the function still inlines
// as efficiently as p+x, but all call sites should use the string to
// record why the addition is safe, which is to say why the addition
// does not cause x to advance to the very end of p's allocation
// and therefore point incorrectly at the next block in memory.
func add(p unsafe.Pointer, x uintptr, whySafe string) unsafe.Pointer {
	return unsafe.Pointer(uintptr(p) + x)
}
