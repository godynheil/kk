package app

import (
	"testing"

	"github.com/godynheil/kk/internal/core"
)

func TestRequiredObjectsFromLive(t *testing.T) {
	live := map[string]core.LiveObject{
		"bbb": {
			OID:  "bbb",
			Size: 20,
			Refs: []core.ObjectRef{{Commit: "c2", Path: "assets/b.bin"}},
		},
		"aaa": {
			OID:  "aaa",
			Size: 10,
			Refs: []core.ObjectRef{{Commit: "c1", Path: "assets/a.bin"}, {Commit: "c3", Path: "copy/a.bin"}},
		},
	}

	got := RequiredObjectsFromLive(live)
	if len(got) != 2 {
		t.Fatalf("expected 2 objects, got %d", len(got))
	}
	if got[0].OID != "aaa" || got[0].Size != 10 || got[0].Path != "assets/a.bin" {
		t.Fatalf("unexpected first object: %+v", got[0])
	}
	if got[1].OID != "bbb" || got[1].Size != 20 || got[1].Path != "assets/b.bin" {
		t.Fatalf("unexpected second object: %+v", got[1])
	}
}
