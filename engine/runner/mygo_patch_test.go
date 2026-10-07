package runner

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// mygoSourceInCache returns the pristine source directory of the pinned mygo
// release from the module cache, skipping the test when it is not present.
func mygoSourceInCache(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("go", "env", "GOMODCACHE").Output()
	if err != nil {
		t.Skipf("go not available: %v", err)
	}
	// Module cache paths escape uppercase letters as !lowercase; the pinned
	// module path has none.
	dir := filepath.Join(strings.TrimSpace(string(out)), "github.com", "egoist", "mygo@"+mygoVersion)
	if _, err := os.Stat(filepath.Join(dir, "internal", "platform", "surface.go")); err != nil {
		t.Skipf("mygo %s not in module cache: %v", mygoVersion, err)
	}
	return dir
}

// TestMygoPatchesApply verifies every anchor matches the pinned release
// exactly once and the resulting tree carries the key mappings.
func TestMygoPatchesApply(t *testing.T) {
	src := mygoSourceInCache(t)
	dst := filepath.Join(t.TempDir(), "mygo-patched")
	if err := copyTree(src, dst); err != nil {
		t.Fatalf("copyTree: %v", err)
	}
	if err := applyMygoPatches(dst); err != nil {
		t.Fatalf("applyMygoPatches: %v", err)
	}

	surface := filepath.Join(dst, "internal", "platform", "surface.go")
	data, err := os.ReadFile(surface)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"KeyPlus\n", "KeyAsterisk\n", "case '+':", "case '*':"} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("patched surface.go missing %q", want)
		}
	}

	keys, err := os.ReadFile(filepath.Join(dst, "ui", "keys.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(keys), "KeyPlus         = Key(platform.KeyPlus)") {
		t.Fatalf("patched ui/keys.go missing KeyPlus:\n%s", keys)
	}

	gtk, err := os.ReadFile(filepath.Join(dst, "internal", "linux", "surface.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"0x2b: platform.KeyPlus", "0xffab: platform.KeyPlus", "0xffae: platform.KeyPeriod"} {
		if !strings.Contains(string(gtk), want) {
			t.Fatalf("patched linux/surface.go missing %q", want)
		}
	}
}
