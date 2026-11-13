package hack

import "unsafe"

// Slice converts string to slice without copy.
// The result aliases s; never modify it and never keep it after the call.
func Slice(s string) (b []byte) {
	return unsafe.Slice(unsafe.StringData(s), len(s))
}
