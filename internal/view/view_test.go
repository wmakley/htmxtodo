package view

import (
	"io/fs"
	"os"
	"testing"
)

func TestFileSystemBehavior(t *testing.T) {
	dirFS := os.DirFS(".")
	dirFS2 := dirFS.(fs.ReadDirFS)

	entries, err := dirFS2.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}

	t.Log(entries)
	if len(entries) == 0 {
		t.Fatal("expected entries")
	}

	_, ok := dirFS.(fs.GlobFS)
	if ok {
		t.Fatal("expected not to be GlobFS")
	}
}
