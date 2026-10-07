package runner

import (
	"bufio"
	"bytes"
	"debug/elf"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

// bundleExcludeRegex lists shared libraries NOT bundled into the AppImage:
//   - glibc core (every target system has a compatible-or-newer glibc; the
//     AppImage cannot ship a portable ld.so anyway)
//   - Mesa/OpenGL driver stack (hardware-specific, present on graphical targets)
//
// Everything else (WebKitGTK, GTK, X11 client libs, codecs, ...) is bundled.
var bundleExcludeRegex = regexp.MustCompile(`/(ld-linux|libc\.so|libm\.so|libpthread|libdl\.so|librt\.so|libresolv|libnsl|libutil|libanl|libGL[^a-z]|libEGL|libGLX|libGLdispatch|libOpenGL|libgbm|libglapi|gallium|mesa|swrast)`)

// Bundle packs a compiled binary plus its shared-library closure into a
// self-contained AppImage at outputPath. The binary itself is built
// beforehand (e.g. by BuildBundle); for CGO programs it stays dynamically
// linked and the AppImage ships the non-glibc, non-Mesa part of `ldd`.
func Bundle(binPath, outputPath string) error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("-bundle 仅支持 Linux（AppImage 格式）")
	}
	absBin, err := filepath.Abs(binPath)
	if err != nil {
		return fmt.Errorf("invalid binary path: %w", err)
	}
	if _, err := os.Stat(absBin); err != nil {
		return fmt.Errorf("binary not found: %s", absBin)
	}

	name := strings.TrimSuffix(filepath.Base(outputPath), ".AppImage")
	workDir, err := os.MkdirTemp("", "codong-bundle-*")
	if err != nil {
		return fmt.Errorf("cannot create bundle temp dir: %w", err)
	}
	defer os.RemoveAll(workDir)

	appDir := filepath.Join(workDir, "AppDir")
	binDir := filepath.Join(appDir, "usr", "bin")
	libDir := filepath.Join(appDir, "usr", "lib")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(libDir, 0755); err != nil {
		return err
	}

	// Copy the program binary.
	if err := copyFile(absBin, filepath.Join(binDir, name)); err != nil {
		return fmt.Errorf("cannot copy binary: %w", err)
	}

	// Collect and copy the shared-library closure.
	libs, err := lddClosure(absBin)
	if err != nil {
		return err
	}
	bundled := 0
	for _, l := range libs {
		if bundleExcludeRegex.MatchString(l.path) {
			continue
		}
		if err := copyFile(l.path, filepath.Join(libDir, l.soname)); err != nil {
			return fmt.Errorf("cannot copy library %s: %w", l.path, err)
		}
		bundled++
	}

	// AppRun launches the binary with the bundled libs first on the search path.
	appRun := fmt.Sprintf(`#!/bin/sh
HERE="$(dirname "$(readlink -f "$0")")"
export LD_LIBRARY_PATH="$HERE/usr/lib:$LD_LIBRARY_PATH"
exec "$HERE/usr/bin/%s" "$@"
`, name)
	if err := os.WriteFile(filepath.Join(appDir, "AppRun"), []byte(appRun), 0755); err != nil {
		return err
	}

	// Desktop entry + icon (appimagetool warns without them).
	desktop := fmt.Sprintf(`[Desktop Entry]
Name=%s
Exec=%s
Type=Application
Categories=Development;Utility;
Icon=%s
Terminal=true
`, name, name, name)
	if err := os.WriteFile(filepath.Join(appDir, name+".desktop"), []byte(desktop), 0644); err != nil {
		return err
	}
	iconDir := filepath.Join(appDir, "usr", "share", "icons", "hicolor", "256x256", "apps")
	if err := os.MkdirAll(iconDir, 0755); err != nil {
		return err
	}
	iconPath := filepath.Join(iconDir, name+".png")
	if err := writeDefaultIcon(iconPath); err != nil {
		return fmt.Errorf("cannot write icon: %w", err)
	}
	// Root-level copy: appimagetool derives .DirIcon from it.
	if err := copyFile(iconPath, filepath.Join(appDir, name+".png")); err != nil {
		return err
	}

	// Run appimagetool (self-extracting mode; no FUSE required). The type2
	// runtime is fetched separately because appimagetool's internal
	// downloader ignores proxy settings.
	tool, err := ensureAppImageTool()
	if err != nil {
		return err
	}
	runtimeFile, err := ensureAppImageRuntime()
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Bundling %d shared libraries into AppImage...\n", bundled)
	cmd := exec.Command(tool, "--runtime-file", runtimeFile, appDir, outputPath)
	cmd.Env = append(os.Environ(), "APPIMAGE_EXTRACT_AND_RUN=1", "ARCH=x86_64")
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("appimagetool failed: %w", err)
	}
	fmt.Fprintf(os.Stderr, "Built: %s\n", outputPath)
	return nil
}

// BuildBundle compiles codFile and packs the result into an AppImage.
func BuildBundle(codFile, outputPath string, static bool) error {
	if !strings.HasSuffix(outputPath, ".AppImage") {
		outputPath += ".AppImage"
	}
	tmpDir, err := os.MkdirTemp("", "codong-bundle-bin-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	binPath := filepath.Join(tmpDir, "program")
	if err := Build(codFile, binPath, static); err != nil {
		return err
	}
	return Bundle(binPath, outputPath)
}

type bundleLib struct {
	soname string
	path   string
}

// lddClosure returns the resolved shared-library dependencies of bin.
// A statically linked binary yields an empty list.
func lddClosure(bin string) ([]bundleLib, error) {
	// Detect static linking from the ELF program headers: ldd's own "not a
	// dynamic executable" message is localized and written to stderr, so it
	// cannot be matched reliably. Only executables with a PT_INTERP need libs.
	ef, err := elf.Open(bin)
	if err != nil {
		return nil, fmt.Errorf("not an ELF binary: %w", err)
	}
	hasInterp := false
	for _, p := range ef.Progs {
		if p.Type == elf.PT_INTERP {
			hasInterp = true
			break
		}
	}
	ef.Close()
	if !hasInterp {
		return nil, nil
	}

	out, lerr := exec.Command("ldd", bin).CombinedOutput()
	var libs []bundleLib
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "linux-vdso") {
			continue
		}
		fields := strings.Fields(line)
		switch {
		case len(fields) >= 3 && fields[1] == "=>":
			if !strings.HasPrefix(fields[2], "/") {
				// "libfoo.so => not found" — the reason text is localized,
				// the "=>" structure is not.
				return nil, fmt.Errorf("missing shared library: %s", fields[0])
			}
			libs = append(libs, bundleLib{soname: fields[0], path: fields[2]})
		case len(fields) >= 1 && strings.HasPrefix(fields[0], "/"):
			libs = append(libs, bundleLib{soname: filepath.Base(fields[0]), path: fields[0]})
		}
	}
	if lerr != nil {
		return nil, fmt.Errorf("ldd failed: %w", lerr)
	}
	return libs, nil
}

// copyFile copies src to dst, dereferencing symlinks.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

// writeDefaultIcon renders a simple 256x256 circle logo so appimagetool
// does not warn about a missing icon.
func writeDefaultIcon(path string) error {
	const size = 256
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	blue := color.RGBA{32, 128, 255, 255}
	c, r := float64(size)/2, float64(size)/2-16
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx, dy := float64(x)-c+0.5, float64(y)-c+0.5
			if dx*dx+dy*dy <= r*r {
				img.Set(x, y, blue)
			}
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

// appImageToolsDir returns ~/.codong/tools (created on demand), where the
// appimagetool and type2-runtime downloads are cached.
func appImageToolsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".codong", "tools")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return dir, nil
}

// ensureAppImageTool downloads the appimagetool AppImage into ~/.codong/tools
// on first use and returns its path.
func ensureAppImageTool() (string, error) {
	toolsDir, err := appImageToolsDir()
	if err != nil {
		return "", err
	}
	toolPath := filepath.Join(toolsDir, "appimagetool-x86_64.AppImage")
	if info, err := os.Stat(toolPath); err == nil && info.Mode()&0111 != 0 {
		return toolPath, nil
	}
	const toolURL = "https://github.com/AppImage/appimagetool/releases/download/continuous/appimagetool-x86_64.AppImage"
	fmt.Fprintf(os.Stderr, "Downloading appimagetool (first time only)...\n")
	if err := downloadFile(toolURL, toolPath); err != nil {
		return "", err
	}
	if err := os.Chmod(toolPath, 0755); err != nil {
		return "", err
	}
	return toolPath, nil
}

// ensureAppImageRuntime downloads the AppImage type-2 runtime stub
// (appended by appimagetool via --runtime-file).
func ensureAppImageRuntime() (string, error) {
	toolsDir, err := appImageToolsDir()
	if err != nil {
		return "", err
	}
	runtimePath := filepath.Join(toolsDir, "runtime-x86_64")
	if _, err := os.Stat(runtimePath); err == nil {
		return runtimePath, nil
	}
	const runtimeURL = "https://github.com/AppImage/type2-runtime/releases/download/continuous/runtime-x86_64"
	fmt.Fprintf(os.Stderr, "Downloading AppImage runtime (first time only)...\n")
	if err := downloadFile(runtimeURL, runtimePath); err != nil {
		return "", err
	}
	return runtimePath, nil
}

func downloadFile(url, dest string) error {
	client := &http.Client{Timeout: 10 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("cannot download %s: %w（如需代理请 export https_proxy=... 后重试）", filepath.Base(url), err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("cannot download %s: HTTP %d", filepath.Base(url), resp.StatusCode)
	}
	f, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
