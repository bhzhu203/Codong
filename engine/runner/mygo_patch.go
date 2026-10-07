package runner

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/codong-lang/codong/engine/goirgen"
)

// The mygo GUI library drops the plus and asterisk keys on its GTK backend
// (no KeyPlus/KeyAsterisk mapping) and its keyval layer cannot see the numpad
// operators at all. Codong therefore pins a mygo release and rebuilds it with
// a small key-mapping patch: every mygo build resolves the module to the
// patched copy made here via a replace directive in go.mod.
const (
	// mygoModule is the module path of the mygo GUI library.
	mygoModule = "github.com/egoist/mygo"

	// mygoVersion is the mygo release Codong pins and patches. Bumping it
	// means re-checking every anchor in mygoPatches against the new release.
	mygoVersion = "v0.2.16"

	// mygoPatchRev names this patch set; it is both the idempotency marker
	// of the patched copy under ~/.codong/pkgs and part of the binary cache
	// key (see cacheEpoch in runner.go).
	mygoPatchRev = "codong1"

	// mygoPatchMarker is the file marking a fully patched copy inside its
	// directory; its content is the patch revision.
	mygoPatchMarker = ".codong-patch"
)

// mygoPatchSpec is one anchored replacement in the mygo sources: old must
// appear exactly once in file, and is replaced by new.
type mygoPatchSpec struct {
	file string
	old  string
	new  string
}

// mygoPatches maps KeyPlus and KeyAsterisk through all three layers of the
// mygo input stack: the platform key enum, KeyForRune and the GTK keyval
// table. Main-keyboard "+" and "*" usually sit behind Shift while the numpad
// sends them bare, so both shapes answer; the numpad digits, "." and Enter
// are mapped too.
var mygoPatches = []mygoPatchSpec{
	{
		file: "internal/platform/surface.go",
		old: `	KeyBack
	KeyForward
)
`,
		new: `	KeyBack
	KeyForward
	// KeyPlus and KeyAsterisk are the keys that type + and *: those on the
	// main keyboard on layouts where they are unshifted, and the numpad ones
	// everywhere.
	KeyPlus
	KeyAsterisk
)
`,
	},
	{
		file: "internal/platform/surface.go",
		old: `// KeyForRune returns the key typing r unshifted, KeyUnknown for others.
func KeyForRune(r rune) Key {`,
		new: `// KeyForRune returns the key typing r, KeyUnknown for others. Plus and
// asterisk answer too: they are keys of their own on the numpad, and the
// main keyboard types them with these keys behind a shift.
func KeyForRune(r rune) Key {`,
	},
	{
		file: "internal/platform/surface.go",
		old: `	case ' ':
		return KeySpace
	}
	return KeyUnknown
`,
		new: `	case ' ':
		return KeySpace
	case '+':
		return KeyPlus
	case '*':
		return KeyAsterisk
	}
	return KeyUnknown
`,
	},
	{
		file: "internal/linux/surface.go",
		old: `	0x1008ff26: platform.KeyBack, 0x1008ff27: platform.KeyForward, // XF86Back, XF86Forward
}
`,
		new: `	0x1008ff26: platform.KeyBack, 0x1008ff27: platform.KeyForward, // XF86Back, XF86Forward
	// The keyvalToUnicode fallback cannot be relied on for the numpad, and
	// plus and asterisk may arrive as themselves: map them all explicitly.
	0x2b: platform.KeyPlus, 0x2a: platform.KeyAsterisk, // plus, asterisk
	0xffab: platform.KeyPlus, 0xffaa: platform.KeyAsterisk, // KP_Add, KP_Multiply
	0xffad: platform.KeyMinus, 0xffaf: platform.KeySlash, // KP_Subtract, KP_Divide
	0xffae: platform.KeyPeriod, // KP_Decimal
	0xffb0: platform.Key0, 0xffb1: platform.Key1, 0xffb2: platform.Key2, 0xffb3: platform.Key3,
	0xffb4: platform.Key4, 0xffb5: platform.Key5, 0xffb6: platform.Key6, 0xffb7: platform.Key7,
	0xffb8: platform.Key8, 0xffb9: platform.Key9, // KP_0 .. KP_9
}
`,
	},
	{
		file: "ui/keys.go",
		old: `	KeyMinus        = Key(platform.KeyMinus)
	KeyEqual        = Key(platform.KeyEqual)
`,
		new: `	KeyMinus        = Key(platform.KeyMinus)
	KeyEqual        = Key(platform.KeyEqual)
	KeyPlus         = Key(platform.KeyPlus)
	KeyAsterisk     = Key(platform.KeyAsterisk)
`,
	},
}

// pkgsDir returns ~/.codong/pkgs, where derived module copies live.
func pkgsDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.TempDir()
	}
	return filepath.Join(home, ".codong", "pkgs")
}

// patchedMygoDir is the directory the patched mygo copy lives in.
func patchedMygoDir() string {
	return filepath.Join(pkgsDir(), fmt.Sprintf("mygo@%s-%s", mygoVersion, mygoPatchRev))
}

// mygoPatchReady reports whether dir holds a fully patched copy of the
// pinned mygo revision.
func mygoPatchReady(dir string) bool {
	data, err := os.ReadFile(filepath.Join(dir, mygoPatchMarker))
	return err == nil && strings.TrimSpace(string(data)) == mygoPatchRev
}

// ensurePatchedMygo returns the directory of the patched mygo copy,
// building it (download pristine, copy, patch, mark) on first use or after
// a patch-revision bump. Idempotent: concurrent or repeated calls converge
// on the same directory.
func ensurePatchedMygo() (string, error) {
	dest := patchedMygoDir()
	if mygoPatchReady(dest) {
		return dest, nil
	}

	src, err := downloadMygoSource()
	if err != nil {
		return "", err
	}

	tmp := fmt.Sprintf("%s.tmp-%d", dest, os.Getpid())
	os.RemoveAll(tmp) // leftover from a crashed run
	if err := copyTree(src, tmp); err != nil {
		os.RemoveAll(tmp)
		return "", fmt.Errorf("mygo 补丁：复制模块源码失败: %w", err)
	}
	if err := applyMygoPatches(tmp); err != nil {
		os.RemoveAll(tmp)
		return "", err
	}
	if err := os.WriteFile(filepath.Join(tmp, mygoPatchMarker), []byte(mygoPatchRev+"\n"), 0644); err != nil {
		os.RemoveAll(tmp)
		return "", fmt.Errorf("mygo 补丁：写入标记失败: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		os.RemoveAll(tmp)
		return "", err
	}

	// Another process may have swapped in its copy while we were building:
	// rename over a non-empty directory fails, and its copy is equivalent.
	os.RemoveAll(dest)
	if err := os.Rename(tmp, dest); err != nil {
		os.RemoveAll(tmp)
		if mygoPatchReady(dest) {
			return dest, nil
		}
		return "", fmt.Errorf("mygo 补丁：安装到 %s 失败: %w", dest, err)
	}
	return dest, nil
}

// downloadMygoSource resolves the pristine mygo source directory through the
// module cache, downloading it first if needed. go mod download wants a
// module context, so it runs in a throwaway one.
func downloadMygoSource() (string, error) {
	dir, err := os.MkdirTemp("", "codong-mygo-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(dir)

	// Match the go directive the real build will use, so toolchain selection
	// happens here exactly as it does later in go mod tidy.
	probe := fmt.Sprintf("module codong-mygo-probe\n\ngo %s\n", goirgen.MygoGoVersion)
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(probe), 0644); err != nil {
		return "", err
	}

	cmd := exec.Command("go", "mod", "download", "-json", mygoModule+"@"+mygoVersion)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("mygo 补丁：下载 %s@%s 失败: %w（离线或受限网络请设置 GOPROXY / https_proxy）", mygoModule, mygoVersion, err)
	}
	var info struct{ Dir string }
	if err := json.Unmarshal(out, &info); err != nil || info.Dir == "" {
		return "", fmt.Errorf("mygo 补丁：无法解析 go mod download 输出: %v", err)
	}
	if _, err := os.Stat(filepath.Join(info.Dir, "ui", "keys.go")); err != nil {
		return "", fmt.Errorf("mygo 补丁：%s 中缺少预期的源码文件: %w", info.Dir, err)
	}
	return info.Dir, nil
}

// copyTree copies the file tree under src into dst, giving every file and
// directory fresh permissions (module cache entries are read-only).
func copyTree(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0644)
	})
}

// applyMygoPatches applies mygoPatches inside root, failing if any anchor
// does not appear exactly once (a sign the pinned release changed).
func applyMygoPatches(root string) error {
	for _, p := range mygoPatches {
		path := filepath.Join(root, p.file)
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("mygo 补丁：无法读取 %s: %w", p.file, err)
		}
		src := string(data)
		if n := strings.Count(src, p.old); n != 1 {
			return fmt.Errorf("mygo 补丁：%s 中锚点出现 %d 次（期望 1 次）——mygo %s 的源码与补丁不匹配", p.file, n, mygoVersion)
		}
		patched := strings.Replace(src, p.old, p.new, 1)
		if err := os.WriteFile(path, []byte(patched), 0644); err != nil {
			return fmt.Errorf("mygo 补丁：写入 %s 失败: %w", p.file, err)
		}
	}
	return nil
}

// goModTemplateMygo is the go.mod for programs using mygo: it pins the
// module and redirects it to the patched copy, so go mod tidy never fetches
// the unpatched original and no go.sum entry is needed for it.
func goModTemplateMygo(patchedDir string) string {
	if strings.ContainsAny(patchedDir, " \t\"") {
		patchedDir = strconv.Quote(patchedDir)
	}
	return fmt.Sprintf(`module codong-app

go %s

require %s %s

replace %s => %s
`, goirgen.MygoGoVersion, mygoModule, mygoVersion, mygoModule, patchedDir)
}
