package packager

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPackagerPackUnpack(t *testing.T) {
	tmp := t.TempDir()
	p := NewPackager(tmp)

	srcDir := filepath.Join(tmp, "testdir")
	os.MkdirAll(srcDir, 0755)
	os.WriteFile(filepath.Join(srcDir, "file.txt"), []byte("hello"), 0644)

	err := p.Pack("test", srcDir)
	if err != nil {
		t.Fatalf("Pack: %v", err)
	}

	dstDir := filepath.Join(tmp, "dst")
	os.MkdirAll(dstDir, 0755)
	err = p.Unpack("test", dstDir)
	if err != nil {
		t.Fatalf("Unpack: %v", err)
	}
}

func TestPackagerList(t *testing.T) {
	tmp := t.TempDir()
	p := NewPackager(tmp)

	srcDir := filepath.Join(tmp, "listdir")
	os.MkdirAll(srcDir, 0755)
	os.WriteFile(filepath.Join(srcDir, "a.txt"), []byte("a"), 0644)

	p.Pack("listtest", srcDir)

	files, err := p.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(files) == 0 {
		t.Error("should list files in archive")
	}
}

func TestPackagerExists(t *testing.T) {
	p := NewPackager(t.TempDir())
	if p.Exists("nonexistent") {
		t.Error("nonexistent archive should not exist")
	}
}

func TestPackagerRemove(t *testing.T) {
	tmp := t.TempDir()
	p := NewPackager(tmp)

	srcDir := filepath.Join(tmp, "removeDir")
	os.MkdirAll(srcDir, 0755)
	os.WriteFile(filepath.Join(srcDir, "data.txt"), []byte("x"), 0644)
	p.Pack("removetest", srcDir)

	err := p.Remove("removetest")
	if err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if p.Exists("removetest") {
		t.Error("archive should be removed")
	}
}
