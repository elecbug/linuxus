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

func TestStoragePathsRelativeToConfigIndependentOfWorkingDirectory(t *testing.T) {
	root := t.TempDir()
	t.Chdir(t.TempDir())
	a := &App{configFile: filepath.Join(root, ".env")}
	if err := os.WriteFile(a.configFile, []byte(`AUTH_SERVICE_MOUNTS_HOST_AUTH_LIST_PATH=data/AUTH_LIST
AUTH_SERVICE_MOUNTS_CONTAINER_AUTH_LIST_PATH=/data/AUTH_LIST
VOLUMES_HOST_VOLUMES=volumes
VOLUMES_HOST_HOMES=volumes/homes
VOLUMES_HOST_SHARE=/external/share
VOLUMES_HOST_READONLY=volumes/readonly
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
		t.Fatal("auth path is not config-relative")
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
	for name, data := range map[string][]byte{"linuxusctl": binary, "AUTH_LIST": []byte("private"), ".env": []byte("private settings"), "disk.img": []byte("private disk")} {
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
	a := &App{configFile: filepath.Join(t.TempDir(), ".env")}
	a.Config.Volumes.Host.Share = "/storage/share/"
	a.Config.Volumes.Host.Readonly = "/storage/unused/../readonly/"
	a.normalizeConfigPaths()
	if a.Config.Volumes.Host.Share != "/storage/share" || a.Config.Volumes.Host.Readonly != "/storage/readonly" {
		t.Fatal("absolute path suffixes would change disk image locations")
	}
}

func TestConfigRejectsMalformedEnv(t *testing.T) {
	for _, tc := range []struct {
		name, contents string
		valid          bool
	}{
		{"auto ensure", "VOLUMES_AUTO_ENSURE=true\n", true},
		{"misspelled key", "VOLUMES_AUTOENSURE=true\n", false},
		{"unknown key", "UNKNOWN=true\n", false},
		{"duplicate key", "VOLUMES_AUTO_ENSURE=true\nVOLUMES_AUTO_ENSURE=false\n", false},
		{"legacy YAML", "volumes:\n  auto-ensure: true\n", false},
		{"invalid bool", "VOLUMES_AUTO_ENSURE=maybe\n", false},
		{"empty file", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			a := &App{configFile: filepath.Join(root, ".env")}
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

func TestMissingConfigSuggestsInit(t *testing.T) {
	a := &App{configFile: filepath.Join(t.TempDir(), ".env")}
	if err := a.LoadConfig(); err == nil || !strings.Contains(err.Error(), "linuxusctl init") {
		t.Fatalf("missing initialization hint: %v", err)
	}
}

func TestMovingExecutableKeepsSameConfigAndStorage(t *testing.T) {
	root := t.TempDir()
	cfgPath := filepath.Join(root, "settings", ".env")
	if err := config.InitFile(cfgPath); err != nil {
		t.Fatal(err)
	}
	// Override only storage paths so this test never accesses host production data.
	data := strings.ReplaceAll(config.DefaultEnv, "/var/lib/linuxus/", "../state/")
	if err := os.WriteFile(cfgPath, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	var first config.Config
	for i := 0; i < 2; i++ {
		a := &App{execPath: filepath.Join(t.TempDir(), "linuxusctl"), configFile: cfgPath}
		t.Chdir(t.TempDir())
		if err := a.LoadConfig(); err != nil {
			t.Fatal(err)
		}
		if a.Config.Volumes.Host.Homes != filepath.Join(root, "state", "volumes", "homes") {
			t.Fatal("storage followed the executable")
		}
		if i == 0 {
			first = a.Config
		} else if !reflect.DeepEqual(first, a.Config) {
			t.Fatal("moving executable changed settings")
		}
		t.Setenv(config.ConfigFileEnv, "/unrelated/.env")
		cmd := a.diskServiceCommand()
		var configs []string
		for _, entry := range cmd.Env {
			if strings.HasPrefix(entry, config.ConfigFileEnv+"=") {
				configs = append(configs, entry)
			}
		}
		if !reflect.DeepEqual(configs, []string{config.ConfigFileEnv + "=" + cfgPath}) {
			t.Fatal("disk service selected another config")
		}
	}
}

func TestConfigSymlinkUsesTargetStorageDirectory(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "configuration", ".env")
	if err := config.InitFile(target); err != nil {
		t.Fatal(err)
	}
	data := strings.ReplaceAll(config.DefaultEnv, "/var/lib/linuxus/", "../state/")
	if err := os.WriteFile(target, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(t.TempDir(), ".env")
	if err := os.Symlink(target, alias); err != nil {
		t.Fatal(err)
	}
	a := &App{configFile: alias}
	if err := a.LoadConfig(); err != nil {
		t.Fatal(err)
	}
	if a.configFile != target || a.Config.AuthService.Mounts.HostAuthListPath != filepath.Join(root, "state", "data", "AUTH_LIST") {
		t.Fatal("config symlink redirected storage")
	}
}
