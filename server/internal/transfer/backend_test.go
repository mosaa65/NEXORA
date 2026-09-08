package transfer

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSplitRemoteDir(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"/", nil},
		{"Movies", []string{"Movies"}},
		{"/Documents/Series/Breaking Bad", []string{"Documents", "Series", "Breaking Bad"}},
		{"a//b///c", []string{"a", "b", "c"}},
	}
	for _, c := range cases {
		if got := splitRemoteDir(c.in); !reflect.DeepEqual(got, c.want) {
			t.Errorf("splitRemoteDir(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestJoinRemotePath(t *testing.T) {
	cases := []struct {
		dir, name, want string
	}{
		{"", "a.txt", "a.txt"},
		{"/Documents", "a.txt", "/Documents/a.txt"},
		{"/Documents/", "a.txt", "/Documents/a.txt"},
		{`E:\Movies`, `a.txt`, `E:\Movies\a.txt`},
		{`E:\Movies\`, `a.txt`, `E:\Movies\a.txt`},
	}
	for _, c := range cases {
		if got := joinRemotePath(c.dir, c.name); got != c.want {
			t.Errorf("joinRemotePath(%q,%q) = %q, want %q", c.dir, c.name, got, c.want)
		}
	}
}

func TestStatSize(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "f.bin")
	if err := os.WriteFile(p, make([]byte, 1234), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := statSize(p); got != 1234 {
		t.Fatalf("statSize = %d, want 1234", got)
	}
	if got := statSize(filepath.Join(dir, "nope")); got != 0 {
		t.Fatalf("statSize(missing) = %d, want 0", got)
	}
}

func TestResolveToolFallsBackToName(t *testing.T) {
	if got := resolveTool("", "ios-tool"); got != "ios-tool" {
		t.Fatalf("resolveTool = %q, want fallback name", got)
	}
}
