package hack

import (
	"bytes"
	"testing"
)

func TestByte(t *testing.T) {
	a := "hello world"

	b := Slice(a)

	if !bytes.Equal(b, []byte("hello world")) {
		t.Fatal(string(b))
	}
}
