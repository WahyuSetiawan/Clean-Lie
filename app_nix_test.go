package main

import (
	"strings"
	"testing"
)

func TestNixPreview(t *testing.T) {
	a := NewApp("test", "test")

	if got := a.NixPreview("gc", "", nil); !strings.Contains(got, "nix-collect-garbage") {
		t.Fatalf("preview GC = %q", got)
	}
	if got := a.NixPreview("delete", "user", []int{719, 718}); !strings.Contains(got, "--delete-generations 718 719") {
		t.Fatalf("preview delete = %q", got)
	}
	if got := a.NixPreview("journal", "500M", nil); !strings.Contains(got, "--vacuum-size=500M") {
		t.Fatalf("preview journal = %q", got)
	}
	if got := a.NixPreview("tidak-ada", "", nil); got != "" {
		t.Fatalf("preview tak dikenal = %q", got)
	}
}
