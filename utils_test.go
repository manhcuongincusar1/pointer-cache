package cache

import "testing"

func TestCycles(t *testing.T) {
	type V struct {
		Z int
		E *V
	}

	v := &V{Z: 25}
	want := DeepSize(v)
	v.E = v // induce a cycle
	got := DeepSize(v)
	if got != want {
		t.Errorf("Cyclic size: got %d, want %d", got, want)
	}
}
