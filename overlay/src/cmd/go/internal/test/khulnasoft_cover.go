package test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

// copyCoverageProfile copies the coverage profile report into
// the output file while restoring the original filenames from
// any overlays that were applied by Khulnsoft.
//
// Normally the coverage profile is copied using in [mergeCoverProfile] using:
func copyCoverageProfile(src, dst string, overlays map[string]string) error {
    srcFile, err := os.Open(src)
    if err != nil {
        return fmt.Errorf("failed to open source file: %w", err)
    }
    defer srcFile.Close()

    dstFile, err := os.Create(dst)
    if err != nil {
        return fmt.Errorf("failed to create destination file: %w", err)
    }
    defer dstFile.Close()

    scanner := bufio.NewScanner(srcFile)
    writer := bufio.NewWriter(dstFile)
    defer writer.Flush()

    for scanner.Scan() {
        line := scanner.Text()
        for orig, overlay := range overlays {
            line = strings.ReplaceAll(line, overlay, orig)
        }
        if _, err := writer.WriteString(line + "\n"); err != nil {
            return fmt.Errorf("failed to write to destination file: %w", err)
        }
    }

    if err := scanner.Err(); err != nil {
        return fmt.Errorf("error reading source file: %w", err)
    }

    return nil
}

var (
	khulnsoftOverlayReverseMap map[string]string
	khulnsoftOnce              sync.Once
)

// initKhulnsoftReverseMap initializes the khulnsoftOverlayReverseMap
// which is a mapping of "[pkg]/[overlay_file].go" -> "[pkg]/[original_file].go"
//
// It does this by:
//  1. Reading the overlay file (fsys.OverlayFile)
//  2. Working out the packages that we're overlaying based on their file paths
//  3. Building a map of "[pkg]/[overlay_file].go" -> "[pkg]/[original_file].go"
func initKhulnsoftReverseMap() error {
	// 1) First read the overlay file
	var overlayJSON fsys.OverlayJSON
	{ // This block is mostly copied from: cmd/go/internal/fsys/fsys.go Init

		// Read the overlay file
		b, err := os.ReadFile(fsys.OverlayFile)
		if err != nil {
			return fmt.Errorf("reading overlay file: %v", err)
		}

		// Parse the overlay file
		if err := json.Unmarshal(b, &overlayJSON); err != nil {
			return fmt.Errorf("parsing overlay JSON: %v", err)
		}
	}

	// 2) Work out the packages that we're overlaying

	// Pkg describes a single package, compatible with the JSON output from 'go list'; see 'go help list'.
	type Pkg struct {
		ImportPath string
		Dir        string
		Error      *struct {
			Err string
		}
	}
	pkgs := make(map[string]*Pkg)

	{ // This block is mostly copied from: cmd/cover/func.go findPkgs
		pkgList := make([]string, 0, len(overlayJSON.Replace))
		pkgSet := make(map[string]struct{})
		for baseFile := range overlayJSON.Replace {
			// We only care about the baseFile, as overlays are reported
			// to be in the original files package
			baseDir := filepath.Dir(canonicalize(baseFile))
			if _, ok := pkgSet[baseDir]; !ok {
				pkgSet[baseDir] = struct{}{}
				pkgList = append(pkgList, baseDir)
			}
		}

		// Now run go list to find the location of every package we care about.
		goTool := filepath.Join(runtime.GOROOT(), "bin/go")
		cmd := exec.Command(goTool, append([]string{"list", "-e", "-json"}, pkgList...)...)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		stdout, err := cmd.Output()
		if err != nil {
			return fmt.Errorf("cannot run go list: %v\n%s", err, stderr.Bytes())
		}
		dec := json.NewDecoder(bytes.NewReader(stdout))
		for {
			var pkg Pkg
			err := dec.Decode(&pkg)
			if err == io.EOF {
				break
			}
			if err != nil {
				return fmt.Errorf("decoding go list json: %v", err)
			}

			if pkg.Error == nil && pkg.ImportPath != "" && pkg.Dir != "" {
				pkgs[pkg.Dir] = &pkg
			}
		}
	}

	// 3) Now build the reverse map
	khulnsoftOverlayReverseMap = make(map[string]string)
	for basePath, overlayPath := range overlayJSON.Replace {
		// Find the package for the original file
		basePath = canonicalize(basePath)
		pkg, found := pkgs[filepath.Dir(basePath)]
		if !found {
			// Some of Khulnsoft's internals are not in the go list, so we ignore them
			continue
		}

		// If the original file and overlay file are reported with different filenames
		// then lets add a mapping to the reverse map.
		reportedFile := filepath.Join(pkg.ImportPath, filepath.Base(canonicalize(overlayPath)))
		originalFile := filepath.Join(pkg.ImportPath, filepath.Base(basePath))
		if reportedFile != originalFile {
			khulnsoftOverlayReverseMap[reportedFile] = originalFile
		}
	}

	return nil
}

// copied from cmd/go/internal/fsys/fsys.go
func canonicalize(path string) string {
	cwd := base.Cwd()

	if path == "" {
		return ""
	}
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}

	if v := filepath.VolumeName(cwd); v != "" && path[0] == filepath.Separator {
		// On Windows filepath.Join(cwd, path) doesn't always work. In general
		// filepath.Abs needs to make a syscall on Windows. Elsewhere in cmd/go
		// use filepath.Join(cwd, path), but cmd/go specifically supports Windows
		// paths that start with "\" which implies the path is relative to the
		// volume of the working directory. See golang.org/issue/8130.
		return filepath.Join(v, path)
	}

	// Make the path absolute.
	return filepath.Join(cwd, path)
}
