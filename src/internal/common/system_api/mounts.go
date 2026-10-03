package system_api

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// mountinfo includes same-filesystem bind mounts that device/inode comparisons miss.
func parseMountPoints(data []byte) ([]string, error) {
	unescape := strings.NewReplacer(`\040`, " ", `\011`, "\t", `\012`, "\n", `\134`, `\`)
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 4096), 1<<20)
	var points []string
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 10 || !strings.Contains(scanner.Text(), " - ") || !filepath.IsAbs(fields[4]) {
			return nil, fmt.Errorf("invalid mountinfo entry")
		}
		points = append(points, filepath.Clean(unescape.Replace(fields[4])))
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read mountinfo: %w", err)
	}
	return points, nil
}

func readMountPoints(path string) (string, []string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", nil, err
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if os.IsNotExist(err) {
		return "", nil, nil
	}
	if err != nil {
		return "", nil, err
	}
	data, err := os.ReadFile("/proc/self/mountinfo")
	if err != nil {
		return "", nil, fmt.Errorf("read mountinfo: %w", err)
	}
	points, err := parseMountPoints(data)
	return resolved, points, err
}

func (LinuxSystemAPI) IsMountPoint(path string) (bool, error) {
	resolved, points, err := readMountPoints(path)
	if err != nil {
		return false, err
	}
	for _, point := range points {
		if point == resolved {
			return true, nil
		}
	}
	return false, nil
}

// MountPointsUnder does not traverse user files or follow symlinks inside them.
// Only the configured root is resolved, allowing deployments on linked storage.
func (LinuxSystemAPI) MountPointsUnder(root string) ([]string, error) {
	resolved, points, err := readMountPoints(root)
	if err != nil || resolved == "" {
		return nil, err
	}
	return mountPointsUnder(resolved, points), nil
}

func mountPointsUnder(root string, points []string) []string {
	var result []string
	for _, point := range points {
		relative, err := filepath.Rel(root, point)
		if err == nil && relative != "." && relative != ".." && !strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
			// Keep duplicate mount points: stacked mounts each need an unmount.
			result = append(result, point)
		}
	}
	return result
}
