package app

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

func listMountedDirsDeepestFirst(root string) ([]string, error) {
	var dirs []string

	if _, err := os.Stat(root); os.IsNotExist(err) {
		return nil, nil
	}

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if path == root {
			return nil
		}
		if !d.IsDir() {
			return nil
		}

		mounted, err := isMountPoint(path)
		if err != nil {
			return nil
		}
		if mounted {
			dirs = append(dirs, path)
			return fs.SkipDir
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(dirs, func(i, j int) bool {
		return len(dirs[i]) > len(dirs[j])
	})

	return dirs, nil
}

func isMountPoint(path string) (bool, error) {
	cmd := exec.Command("mountpoint", "-q", path)
	err := cmd.Run()
	if err == nil {
		return true, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return false, nil
	}
	return false, fmt.Errorf("failed to check mountpoint %s: %w", path, err)
}

func findLoopDevicesForImages(homesDir string) ([]string, error) {
	entries, err := os.ReadDir(homesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read homes dir: %w", err)
	}

	// homes/*.img
	imageSet := make(map[string]struct{})
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".img") {
			fullPath := filepath.Join(homesDir, name)
			absPath, err := filepath.Abs(fullPath)
			if err != nil {
				continue
			}
			imageSet[absPath] = struct{}{}
		}
	}

	if len(imageSet) == 0 {
		return nil, nil
	}

	out, err := runCmdOutput("sudo", "losetup", "-a")
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, nil
		}
		return nil, err
	}

	var loopDevs []string
	seen := make(map[string]struct{})

	scanner := bufio.NewScanner(strings.NewReader(out))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// e.g.: /dev/loop0: []: (/path/to/file.img)
		colonIdx := strings.Index(line, ":")
		lparenIdx := strings.LastIndex(line, "(")
		rparenIdx := strings.LastIndex(line, ")")

		if colonIdx <= 0 || lparenIdx < 0 || rparenIdx < 0 || rparenIdx <= lparenIdx {
			continue
		}

		dev := strings.TrimSpace(line[:colonIdx])
		backingFile := strings.TrimSpace(line[lparenIdx+1 : rparenIdx])

		absBacking, err := filepath.Abs(backingFile)
		if err != nil {
			continue
		}

		if _, ok := imageSet[absBacking]; ok {
			if _, exists := seen[dev]; !exists {
				seen[dev] = struct{}{}
				loopDevs = append(loopDevs, dev)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to parse losetup output: %w", err)
	}

	sort.Strings(loopDevs)
	return loopDevs, nil
}
