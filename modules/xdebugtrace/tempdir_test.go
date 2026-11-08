package xdebugtrace

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"testing"
)

func TestTempDir_RemovesDeadProcessDirs(t *testing.T) {
	base := t.TempDir()

	// A pid that is surely gone: a finished child.
	cmd := exec.Command(os.Args[0], "-test.run=^$")
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	dead := cmd.Process.Pid
	alive := os.Getppid() // the go test runner

	mk := func(name string) {
		if err := os.MkdirAll(filepath.Join(base, name, "sub"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	mk(tempPrefix + strconv.Itoa(dead))
	mk(tempPrefix + strconv.Itoa(alive))
	mk(tempPrefix + strconv.Itoa(os.Getpid())) // own leftover
	mk(tempPrefix + "abc")
	mk("other-dir")

	dir, err := TempDir(base)
	if err != nil {
		t.Fatal(err)
	}
	if dir != filepath.Join(base, tempPrefix+strconv.Itoa(os.Getpid())) {
		t.Errorf("dir = %s", dir)
	}
	if _, err := os.Stat(filepath.Join(dir, "sub")); !os.IsNotExist(err) {
		t.Error("own leftover dir not emptied")
	}
	entries, _ := os.ReadDir(base)
	var got []string
	for _, e := range entries {
		got = append(got, e.Name())
	}
	sort.Strings(got)
	want := []string{tempPrefix + "abc", tempPrefix + strconv.Itoa(alive), tempPrefix + strconv.Itoa(os.Getpid()), "other-dir"}
	sort.Strings(want)
	if len(got) != len(want) {
		t.Fatalf("left %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("left %v, want %v", got, want)
		}
	}
}
