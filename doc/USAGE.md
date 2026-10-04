# 🚀 Usage

## 0. Clone Repository

```bash
git clone https://github.com/elecbug/linuxus
cd linuxus
```

---

## 1. Install Dependencies

### Go (build machine only)

```bash
sudo snap install go --classic
```

### Docker

```bash
sudo apt install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
```

---

## 2. Build Controller

Build the single Linux executable (Go 1.26.1 or newer):

```bash
./shell/build_ctl.sh
```

The script can be invoked from any working directory. It builds a static Linux
binary for the current architecture; set `GOARCH=arm64` to cross-compile.
`linuxusctl` includes the Auth and Manager service modes, Dockerfiles, user
startup script, and web assets. No Go compiler or source tree is required at runtime.

Generated executable:

```bash
./linuxusctl help
```

For a more convenient CLI experience, enable bash completion support.

Run the following command:

```bash
source ./shell/linuxus-completion.bash
```

After enabling completion, you can use `TAB` to automatically complete commands and options.

---

## 3. Run Service

The executable can live anywhere. For example, install it on the system PATH:

```bash
sudo install -m 0755 linuxusctl /usr/local/bin/linuxusctl
sudo linuxusctl init
```

For an existing deployment, use the migration procedure below before running
`init`. For a new deployment, `init` writes the embedded defaults to
`/etc/linuxus/.env` with file permissions `0600`, creating its directory with
mode `0700` when needed. It refuses to overwrite an existing file or symlink.
The source template is `src/internal/common/config/default.env`; rebuild to
include template changes. Runtime configuration is not used during builds.

```text
/etc/linuxus/
└── .env
/var/lib/linuxus/
├── data/
│   ├── AUTH_LIST
│   └── .disk-service/
└── volumes/
```

All commands select `/etc/linuxus/.env` by default, regardless of the current
working directory or executable location. Copying, renaming, or linking the
binary does not select another installation's data. Default storage paths are
absolute paths under `/var/lib/linuxus`; `up` creates missing data directories.

To select a separate configuration, supply an absolute `LINUXUS_CONFIG` path:

```bash
sudo env LINUXUS_CONFIG=/srv/classroom/.env /path/to/linuxusctl init
sudo env LINUXUS_CONFIG=/srv/classroom/.env /path/to/linuxusctl up
```

This selects the configuration only; change its host storage settings as well
if a separate data location is desired. Relative host paths are resolved against
the selected configuration's directory (the target directory for a config
symlink), never against the working directory or executable. The disk service
inherits the exact configuration selected by the parent CLI.

Edit `/etc/linuxus/.env`, then start services from any location:

```bash
sudo linuxusctl up
```

The runtime host needs Linux, a local Docker daemon, `mkfs.ext4` (e2fsprogs),
and `losetup` (util-linux). Disk operations require root privileges. Images use
the executable's architecture. The first image build needs access to the base
images and Ubuntu/Debian package repositories; it does not download Go modules.

Before starting, set `AUTH_SERVICE_SECURITY_SESSION_SECRET` and
`MANAGER_SERVICE_SECURITY_SESSION_SECRET` in `/etc/linuxus/.env` to your own values. `up` creates a missing auth file without
overwriting existing accounts,
prepares shared disks and registered users' disks, and starts the services.
A fresh auth file has no accounts; `MANAGER_SERVICE_ADMIN_ID` designates an account's privileges
and does not create the account automatically.

Host directories can also point to external storage through absolute `.env`
paths or symlinks beneath `/var/lib/linuxus`. Manager passes the
resolved host paths to Docker; it does not need its own copies of the volumes.
The home, shared, and readonly host paths must not overlap or resolve to the
filesystem root. Keep `AUTH_LIST` outside these disk directories and images.
Connection timeouts must be positive; a cleanup timeout of `0s` disables idle
cleanup, and disk sizes must be greater than 1 MiB.
Settings use uppercase keys with underscores, for example
`VOLUMES_AUTO_ENSURE=true` and `AUTH_SERVICE_CONTAINER_EXTERNAL_PORT=8080`.
Unknown or duplicate keys are rejected, and booleans must be `true` or `false`.
Blank lines, comments, optional `export`, and single/double quoted values are
supported. An unquoted `#` starts a comment at the beginning of a value or after
whitespace. Single quotes preserve literal values; double quotes support
`\n`, `\r`, `\t`, `\\`, `\"`, and `\$`. Values occupy one physical line;
variable expansion and shell command execution are not performed. Process
environment variables do not override individual `.env` settings;
`LINUXUS_CONFIG` selects which configuration file to read. Missing settings are
validated normally; they are not filled from embedded defaults at runtime.

Auth route paths must be distinct, clean paths
without leading/trailing slashes or wildcard patterns; `static/` and
`favicon.ico` are reserved. Nested paths such as `auth/login` are supported.
Docker subnet allocation checks all existing Docker IPAM ranges to avoid
conflicts with other projects. Trusted proxy CIDRs support both IPv4 and IPv6.

When HTTPS terminates at a reverse proxy, include its source address in
`AUTH_SERVICE_SECURITY_TRUSTED_PROXIES` and configure it to overwrite
`X-Forwarded-Proto` with the external scheme (`https` or `http`). Auth then sets
`Secure` on HTTPS session cookies and preserves that scheme for the terminal.
The Auth session cookie is removed before forwarding to a user's container.

The legacy `MANAGER_SERVICE_CONTAINER_HOMES_DIR`,
`MANAGER_SERVICE_CONTAINER_SHARE_DIR`, and `MANAGER_SERVICE_CONTAINER_READONLY_DIR`
settings are retained for configuration compatibility but are no longer mounted
into Manager.

### 3.1 Start / Manage Services

```bash
./linuxusctl <OPTION>
```

### ⚙️ Available Options

| Command                           | Description                                                                                      |
| --------------------------------- | ------------------------------------------------------------------------------------------------ |
| `init`                            | Create `/etc/linuxus/.env` using embedded defaults; preserve existing files              |
| `help`                            | Show help message                                                                                |
| `up`                              | Build images and start services                                                                  |
| `down`                            | Stop and remove services                                                                         |
| `restart`                         | Restart services                                                                                 |
| `ps [OPTION]`                     | Show status about linuxus service                                                                |
| `add-user --user <USERNAME>`      | Add a new user                                                                                   |
| `remove-user --user <USERNAME>`   | Remove an existing user                                                                          |
| `clean-volume <OPTION>`           | Remove all user directories if the option is all, otherwise remove specific user directory       |
| `ensure-disk <OPTION>`            | Create a missing user directory if the option is all, otherwise create a specific user directory |

---

### 3.2 Example Usage

```bash
sudo ./linuxusctl init            # Create /etc/linuxus/.env once
sudo ./linuxusctl up               # Assemble images, prepare disks and start
sudo ./linuxusctl restart          # Restart
./linuxusctl ps network            # Show network status of linuxus service
```

---

## 4. Setup Authentication (Signup-based)

> [!NOTE]
> Linuxus provides a built-in signup system via the web interface.

---

### 4.1 Enable Signup

Edit `/etc/linuxus/.env`:

```dotenv
AUTH_SERVICE_ALLOW_SIGNUP=true
```

A **Signup** link will appear on the login page.

---

Session cookies expire after 12 hours and are checked against the current account
on each authenticated request. Removing or changing an account invalidates its
previous cookies. Upgrading from the older cookie format requires users to log in
again once. CLI and Auth account updates use the same file lock while preserving
the auth file's Docker bind mount.

### 4.2 User Registration Flow

1. User opens the login page
2. Clicks **Signup**
3. Enters ID and password
4. Account is registered

With `VOLUMES_AUTO_ENSURE=true`, the first shell access prepares the newly registered user's disk automatically. With the setting omitted or false, initialize new users manually as shown below.

---

### 4.3 Activate User Environment

Enable automatic disk activation in `/etc/linuxus/.env`:

```dotenv
VOLUMES_AUTO_ENSURE=true
```

Apply the setting with `sudo ./linuxusctl restart`. `up` starts the private host
`linuxusctl serve-disks` process; `down` stops it. Manager requests disk preparation
over a Unix socket before creating or restarting a user container. Disk errors
prevent the shell from starting, and existing images are reused rather than
formatted again. Manual `ensure-disk` remains available.

The host process runs with the mounting privileges of `sudo linuxusctl up`.
Its socket has mode 0600 in a mode-0700 `.disk-service` directory beside `AUTH_LIST`.
Only Manager receives that directory as a read-only mount. There is no TCP
listener. Requests accept a registered user ID only; paths and disk limits come
from the deployment configuration. Manager's container capabilities are unchanged.
The default service log is `/var/lib/linuxus/data/.disk-service/service.log`. Run `sudo ./linuxusctl up`
after a host reboot to restore mounts and the host process; Docker container
restart policies alone do not restart this process. As with the existing deployment,
Docker must run locally on the Linux host.

When `VOLUMES_AUTO_ENSURE` is omitted or false, users registered after startup need manual
activation:

```bash
sudo ./linuxusctl ensure-disk --user <USERNAME>
```

or initialize all missed user environments:

```bash
sudo ./linuxusctl ensure-disk --all
```

This step:

* Creates home directories
* Mounts volumes
* Activates user accounts

---

### 4.4 Admin Account

* Default admin ID: `alpha`
* Configurable in:

```dotenv
MANAGER_SERVICE_ADMIN_ID=alpha
```

---

## 5. Volume Structure

```
/var/lib/linuxus/volumes/
├── homes/
│   ├── alice.img      # ext4 disk image
│   └── alice/         # mounted home
├── share.img
├── share/
├── readonly.img
└── readonly/
```

---

Containers may be removed and recreated while these disk images retain data.
After a host reboot, run `sudo ./linuxusctl up` to remount them before using the
service. `down` removes containers and networks but preserves disks.
`clean-volume` removes data; it stops on unmount or loop-device errors rather
than continuing deletion. Cleaning all volumes leaves unrelated files in the
volume root untouched.

### Migrating the previous executable-relative .env/data/volumes layout

The CLI now reads `/etc/linuxus/.env` rather than an adjacent `.env`. Existing
live storage must be stopped and unmounted before moving it. Keep a backup before
migration; `clean-volume` must not be used because it deletes disk contents.

For the standard previous layout, use the provided one-time migration helper
with the newly built CLI:

```bash
sudo python3 ./shell/migrate_system_paths.py --from /absolute/old/deployment
sudo ./linuxusctl up
```

If the new binary is elsewhere, add `--ctl /absolute/path/to/linuxusctl`.
The helper supports ordinary `data/` and `volumes/` directories on the same
filesystem as `/var/lib`. It refuses custom layouts, existing destination data,
and cross-filesystem moves before stopping services. Python 3 is needed only
for this migration helper, not for runtime operation.

It uses the old `.env` to stop managed containers and the disk service, locks disk
operations, unmounts nested volumes, detaches their loop devices, and verifies
that mounts and loop attachments are gone. It then renames the directories to
`/var/lib/linuxus`, preserving images, ownership, sparse allocation and inodes.
Existing `data/` and `volumes/` paths become compatibility symlinks. Configuration
is published to `/etc/linuxus/.env` with only the five host storage paths changed;
other settings, including session secrets, are preserved. A failed configuration
publication attempts to restore the original directory locations. Services remain
stopped until the subsequent `up` command remounts storage and starts them.

For custom storage or another filesystem, perform the migration manually:

1. Stop services with the old configuration explicitly selected:
   `sudo env LINUXUS_CONFIG=/absolute/old/.env /path/to/linuxusctl down`.
2. Unmount all managed user and shared mounts, deepest first, and detach their
   loop devices. Verify that none remain before copying images.
3. Copy `data/` and `volumes/` to a fresh `/var/lib/linuxus` directory, preserving
   ownership, permissions, symlinks and sparse files (for example, `cp -a --sparse=always`).
   Verify the copies before removing any originals.
4. Copy the original `.env` to `/etc/linuxus/.env` with permissions `0600` and
   update the five host storage paths to their new absolute locations.
5. Run `sudo /path/to/linuxusctl up` and check accounts and existing home data.

### Migrating from config.yml

YAML configurations are no longer loaded. For an older YAML installation,
convert its settings to `.env` and apply the same storage migration sequence.
Nested YAML names map to uppercase underscore-separated keys, with hyphens also
replaced by underscores. For example, `volumes.auto-ensure` becomes
`VOLUMES_AUTO_ENSURE`, and `auth_service.security.session_secret` becomes
`AUTH_SERVICE_SECURITY_SESSION_SECRET`. Preserve session secrets and transfer
existing images rather than starting with an empty auth list or fresh disks.

### Migrating an existing source-relative deployment

Existing `src/data` and `src/volumes` are not moved automatically. Before changing
paths, stop the old services and back up `AUTH_LIST` and all disk images.

To keep existing storage in place, configure its absolute host paths in the new
`.env`, or link the deployment's `data/` and `volumes/` to those locations. Keep the
storage directories even if you remove the rest of the source checkout.

To relocate storage completely, stop the services, unmount the old user/shared
disks, and detach their loop devices before moving the directories. Copy the
`AUTH_LIST` and `.img` files into `/var/lib/linuxus/data/` and `/var/lib/linuxus/volumes/`, preserving
ownership and permissions. Then run `sudo ./linuxusctl up` to attach and mount
the existing images at the new paths. Do not use `clean-volume` for migration;
it deletes the images and their contents.

---

## 6. Directory Permissions

### 👤 User (`homes/<USER>`)

* Private
* Mounted to `/home/<linux_username>` inside each user container (default: `/home/user`)

### 📂 Share

* RWX for all users
* `/home/share`

### 🔒 Readonly

* Read/execute for users
* Write for admin only
* `/home/readonly`

---

## 🌐 APPENDIX - Preview Image

> ![](./fig/04-arch.png)
> Linuxus Architecture Diagram

> ![](./fig/01-login.png)
> Login Page

> ![](./fig/02-shell_1.png)
> Shell Page - Access

> ![](./fig/03-shell_2.png)
> Shell Page - Test GCC
