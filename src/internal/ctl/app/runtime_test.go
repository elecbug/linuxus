package app

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/elecbug/linuxus/src/internal/common/config"
	"github.com/elecbug/linuxus/src/internal/common/diskservice"
)

func TestDeploymentPathsIndependentOfWorkingDirectory(t *testing.T) {
	root := t.TempDir()
	t.Chdir(t.TempDir())
	a := &App{runtimeRoot: root, configFile: filepath.Join(root, "config.yml")}
	if err := os.WriteFile(a.configFile, []byte(`auth_service:
  mounts:
    host_auth_list_path: data/AUTH_LIST
    container_auth_list_path: /data/AUTH_LIST
volumes:
  host:
    volumes: volumes
    homes: volumes/homes
    share: /external/share
    readonly: volumes/readonly
`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := a.LoadConfig(); err != nil {
		t.Fatal(err)
	}
	if len(a.UserIDs) != 0 {
		t.Fatal("fresh deployment has users")
	}
	if a.Config.Volumes.Host.Homes != filepath.Join(root, "volumes/homes") {
		t.Fatal(a.Config.Volumes.Host.Homes)
	}
	if a.Config.Volumes.Host.Share != "/external/share" {
		t.Fatal("absolute path changed")
	}
	if a.Config.AuthService.Mounts.HostAuthListPath != filepath.Join(root, "data/AUTH_LIST") {
		t.Fatal("auth path is not deployment-relative")
	}
	if _, err := os.Stat(filepath.Join(root, "data")); !os.IsNotExist(err) {
		t.Fatal("loading configuration changed disk state")
	}

	authSpec, err := a.buildAuthRuntimeSpec()
	if err != nil {
		t.Fatal(err)
	}
	wantBind := filepath.Join(root, "data/AUTH_LIST") + ":/data/AUTH_LIST:rw"
	if !reflect.DeepEqual(authSpec.Volumes, []string{wantBind}) {
		t.Fatal(authSpec.Volumes)
	}
	managerSpec, err := a.buildManagerRuntimeSpec()
	if err != nil {
		t.Fatal(err)
	}
	var transported config.Config
	for _, env := range managerSpec.Environment {
		if strings.HasPrefix(env, "ENV=") {
			if err := json.Unmarshal([]byte(strings.TrimPrefix(env, "ENV=")), &transported); err != nil {
				t.Fatal(err)
			}
		}
	}
	if transported.Volumes.Host.Homes != a.Config.Volumes.Host.Homes {
		t.Fatal("manager lost host mount source")
	}
	if managerSpec.Privileged {
		t.Fatal("manager should only need the Docker socket")
	}
}

func TestRuntimeBuildContextsContainOnlyDeploymentAssets(t *testing.T) {
	root := t.TempDir()
	executable := filepath.Join(root, "linuxusctl")
	binary := []byte("test executable")
	for name, data := range map[string][]byte{"linuxusctl": binary, "AUTH_LIST": []byte("private"), "disk.img": []byte("private disk")} {
		if err := os.WriteFile(filepath.Join(root, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	t.Chdir(t.TempDir())
	for _, name := range []string{"auth", "manager", "user"} {
		t.Run(name, func(t *testing.T) {
			reader, err := tarRuntimeContext(name+".Dockerfile", executable)
			if err != nil {
				t.Fatal(err)
			}
			entries := map[string][]byte{}
			tr := tar.NewReader(reader)
			for {
				header, err := tr.Next()
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatal(err)
				}
				data, err := io.ReadAll(tr)
				if err != nil {
					t.Fatal(err)
				}
				entries[header.Name] = data
				if header.Name == "linuxusctl" && header.Mode != 0755 {
					t.Fatal("binary is not executable")
				}
			}
			if len(entries) != 2 {
				t.Fatalf("unexpected context: %v", entries)
			}
			if bytes.Contains(entries["Dockerfile"], []byte("go build")) {
				t.Fatal("runtime still builds source")
			}
			if name == "user" {
				if len(entries["start.sh"]) == 0 {
					t.Fatal("startup script missing")
				}
			} else if !bytes.Equal(entries["linuxusctl"], binary) {
				t.Fatal("binary missing or changed")
			}
		})
	}
}

func TestAutoEnsureManagerSpec(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		a := &App{}
		a.Config.Volumes.AutoEnsure = enabled
		a.Config.AuthService.Mounts.HostAuthListPath = filepath.Join(t.TempDir(), "AUTH_LIST")
		spec, err := a.buildManagerRuntimeSpec()
		if err != nil {
			t.Fatal(err)
		}
		want := []string{"/var/run/docker.sock:/var/run/docker.sock:rw"}
		if enabled {
			want = append(want, a.diskServiceDir()+":"+diskservice.ContainerDir+":ro")
		}
		if !reflect.DeepEqual(spec.Volumes, want) || spec.Privileged {
			t.Fatalf("invalid manager spec: %+v", spec)
		}
		var cfg config.Config
		if err := json.Unmarshal([]byte(strings.TrimPrefix(spec.Environment[0], "ENV=")), &cfg); err != nil {
			t.Fatal(err)
		}
		if cfg.Volumes.AutoEnsure != enabled {
			t.Fatal("auto-ensure was lost in transport")
		}
	}
}

func TestAbsoluteHostPathsAreCleanedBeforeDiskOperations(t *testing.T) {
	a := &App{runtimeRoot: t.TempDir()}
	a.Config.Volumes.Host.Share = "/storage/share/"
	a.Config.Volumes.Host.Readonly = "/storage/unused/../readonly/"
	a.normalizeConfigPaths()
	if a.Config.Volumes.Host.Share != "/storage/share" || a.Config.Volumes.Host.Readonly != "/storage/readonly" {
		t.Fatal("absolute path suffixes would change disk image locations")
	}
}

func TestConfigRejectsTyposAndMultipleDocuments(t *testing.T) {
	for _, tc := range []struct {
		name, contents string
		valid          bool
	}{
		{"hyphenated key", "volumes:\n  auto-ensure: true\n", true},
		{"misspelled key", "volumes:\n  auto_ensure: true\n", false},
		{"unknown section", "unknown: true\n", false},
		{"duplicate documents", "volumes:\n  auto-ensure: true\n---\nvolumes:\n  auto-ensure: false\n", false},
		{"trailing empty document", "volumes: {}\n---\n", false},
		{"empty file", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			a := &App{runtimeRoot: root, configFile: filepath.Join(root, "config.yml")}
			if err := os.WriteFile(a.configFile, []byte(tc.contents), 0600); err != nil {
				t.Fatal(err)
			}
			err := a.LoadConfig()
			if (err == nil) != tc.valid {
				t.Fatalf("accepted=%t, want %t: %v", err == nil, tc.valid, err)
			}
			if tc.valid && !a.Config.Volumes.AutoEnsure {
				t.Fatal("auto-ensure was not loaded")
			}
		})
	}
}
