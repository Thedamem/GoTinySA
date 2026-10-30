package version

import "testing"

func TestStringNotEmpty(t *testing.T) {
	if got := String(); got == "" {
		t.Fatal("version string must not be empty")
	}
}
