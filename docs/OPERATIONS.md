# Operations

Existing `.env` files remain supported. Omitting the new settings leaves the global
runtime and queue limits disabled and uses the default user image. Before using
these features, run `up` with the new `linuxusctl` to rebuild the Auth and Manager
images. Updating running services disconnects active terminals; schedule upgrades
outside class hours.

## Configuration checks and diagnostics

```bash
sudo linuxusctl config-check
sudo linuxusctl doctor
```

`config-check` validates configuration syntax, values, and paths without connecting
to Docker or reading account records. It prints the selected configuration and
storage paths without exposing secrets. `doctor` checks configuration, the account
file, Docker connectivity, required tools, disk mounts, filesystem free space,
Auth and Manager containers, and the host disk service. Failures return exit code
1; warnings alone return 0. Missing directories or mounts before startup are
reported as warnings. It does not test the external HTTPS proxy or browser login.
Both commands leave files, mounts, and containers unchanged.

## Recovery after reboots and process failures

`supervise` prepares disks before starting Manager and Auth. Every five seconds,
it checks the disk service and service containers and restarts exited processes.
systemd also restarts the supervisor after an abnormal exit. Cached images avoid
rebuilding on every boot. Supervised containers do not use Docker restart
policies, ensuring services cannot start before their disks are ready.

For the initial installation, prepare `/etc/linuxus/.env`, then run:

```bash
sudo install -m 0755 ./linuxusctl /usr/local/bin/linuxusctl
sudo linuxusctl up       # Build service images containing the new binary first.
sudo linuxusctl down
linuxusctl systemd-unit > /tmp/linuxus.service
sudo install -m 0644 /tmp/linuxus.service /etc/systemd/system/linuxus.service
sudo systemctl daemon-reload
sudo systemctl enable --now linuxus.service
```

To use another configuration, set `LINUXUS_CONFIG=/absolute/path/.env` when
generating the unit. The generated unit records absolute configuration and binary
paths. Regenerate it if the binary moves. The host must provide Docker and systemd.

```bash
sudo systemctl status linuxus
sudo journalctl -u linuxus -f
sudo systemctl restart linuxus
sudo systemctl stop linuxus
```

While supervised, use systemctl for service startup, shutdown, and restart.
To upgrade: stop `linuxus` with systemctl, install the binary, run `linuxusctl up`
and `linuxusctl down`, then start `linuxus` with systemctl. Account and disk
commands remain available while supervised. The host disk service runs regardless
of automatic activation settings to support diagnostics, usage, and capacity
checks. With `VOLUMES_AUTO_ENSURE=false`, new user disks still require manual
preparation.

## Session resynchronization

In addition to reporting session starts and stops, Auth sends a complete session
snapshot every 15 seconds. The next snapshot repairs missed disconnect reports.
Snapshots apply atomically, and delayed reports cannot revive sessions absent
from a newer snapshot. Manager discovers existing user containers at startup and every 30 seconds,
granting newly discovered environments a 90-second synchronization grace period.
Session counts expire after two minutes without an update; the configured idle
cleanup timeout then applies. Setting that timeout to `0s` still disables idle
cleanup.

## Account operations that preserve data

```bash
sudo linuxusctl list-users
sudo linuxusctl lock-user --user alice
sudo linuxusctl unlock-user --user alice
sudo linuxusctl reset-password --user alice
sudo linuxusctl disconnect-user --user alice
```

Locking blocks new logins and runtime starts and stops the running environment.
Unlocking allows access again. Disconnecting invalidates existing cookies and
terminal connections while allowing subsequent logins. Password resets use hidden
terminal input with confirmation; passwords are not passed as command arguments.
These commands preserve home images and exercise files. Use `lock-user` to suspend
access; `remove-user` deletes both the account and its disk.

Every account change records a new session generation, so unlocking does not
reactivate old cookies. Legacy bcrypt records remain readable; only updated rows
receive versioned account metadata. Auth and the CLI share file locks and preserve
the inode used by Docker's bind mount. Older binaries cannot read the new record
format; do not downgrade after updating accounts. If a CLI stop request fails, the
account change remains in effect and the error explains that partial result.
Auth also periodically revalidates existing connections.

## Per-user backups and restores

```bash
sudo linuxusctl backup-user --user alice --output /srv/backups/alice.tar.gz
linuxusctl verify-backup --file /srv/backups/alice.tar.gz
sudo linuxusctl restore-user --user alice --file /srv/backups/alice.tar.gz --replace
```

Create the backup directory first. Backups contain the user's ext4 home image and
a manifest recording its SHA-256 checksum, user ID, UID/GID, size, and creation
time. Shared disks, account passwords, and deployment settings are not included.
Store backups outside managed disks on the host. Existing backup files are never
overwritten.

Backup maintenance marks the account unavailable, stops its environment, and
removes its mounts and loop attachments. Success remounts the disk and restores
the original account lock state. The container remains stopped; the user logs in
again to resume. Unlock and password-change requests are rejected during
maintenance. Concurrent backups, restores, and volume cleanup are also rejected. Restore
verification holds this lock before allocating the temporary image.

Restores validate a temporary image before stopping the user's environment.
Validation checks the two permitted archive entries, checksum, size, user ID,
and UID/GID; archive paths are never used as host paths. Empty regions are restored
as sparse files. The target account must already exist with matching UID/GID.
Replacing an existing image requires `--replace`. The original inode is preserved
at `homes/.restore-<user>.previous` before the replacement is published atomically.
If mounting the replacement fails, the original image is restored. Recovery errors
identify any retained previous image. Startup skips accounts under maintenance
and never initializes a missing home for them.

After a failed or interrupted operation, the account remains under maintenance.
Inspect the error, retained images, and `doctor` output, then recover the account.
`recover-user` stops the environment, restores a retained original when present,
and mounts the existing image before clearing maintenance. It refuses missing or
invalid images and leaves account access locked until explicitly unlocked:

```bash
sudo linuxusctl recover-user --user alice  # Recover only when maintenance is inactive; leave the account locked.
sudo linuxusctl ensure-disk --user alice
sudo linuxusctl unlock-user --user alice
```

Checksums detect corruption but do not authenticate the backup's author.
Administrators must control backup files and their storage locations. Restore
space checks conservatively use the image's logical size.

## Global resource limits

The following values are examples. New installations default to no global limits.

```dotenv
CAPACITY_MAX_RUNNING=30
CAPACITY_MAX_PENDING=20
CAPACITY_MIN_FREE_SPACE='5g'
```

The runtime limit counts running user containers, including the administrator's
environment. Reconnecting to an already running environment does not consume
another slot. Preparation requests run serially; the pending limit includes the
request currently being prepared. Excess requests receive HTTP 503 with retry
guidance. Cancelled or expired requests leave the queue.

Host free space is checked on the filesystems containing home and shared images
before new environments, new images, and restores are prepared. This does not
reserve space for every subsequent write or enforce a continuous write limit;
continue monitoring disk usage. Restart services after changing settings.

## Classroom environment templates

The binary includes `linux`, `c`, and `python` templates with introductory Linux
exercises, `hello.c`, and `hello.py`. They use tools in the default user image.
Assign one directly with `assign-template --user alice --template python`.

For additional packages, first build an image based on the default user image
with the required packages and seed files. Run `linuxusctl templates` to see the
default image name. Preserve the existing user UID/GID and `start.sh`/ttyd startup
contract. Configure image tags explicitly; arbitrary images are not pulled
automatically.

```dotenv
USER_SERVICE_TEMPLATES='{"python":{"image":"classroom-python:2026","seed":"/opt/classroom/python"},"c":{"image":"classroom-c:2026","seed":"/opt/classroom/c"}}'
USER_SERVICE_CLASSES='{"class-a":"python","class-b":"c"}'
```

```bash
sudo linuxusctl templates
sudo linuxusctl assign-class --user alice --class class-a
sudo linuxusctl assign-template --user bob --template c
sudo linuxusctl assign-template --user bob --template default
```

Assigning a class clears the individual template selection. An individual
selection takes precedence over the class. Assignment changes stop the existing
environment; the next connection creates it using the selected image. Home data
is preserved. Files in the image's `seed` directory are copied into the home
without overwriting existing files. `default` selects the original default image.
Restart an environment to apply a changed template image.

## Administrator web interface

When the account named by `MANAGER_SERVICE_ADMIN_ID` logs in, the service page
shows an **Admin** link. The interface defaults to `/admin`. Set its public base
path in `.env`, without leading or trailing slashes:

```dotenv
AUTH_SERVICE_SERVICE_URL_ADMIN='ops/classroom'
```

This example serves the page at `/ops/classroom`, its API at
`/ops/classroom/api/users`, and its assets below `/ops/classroom/assets/`.
Navigation links and browser requests follow the configured path. Omitted or
empty values keep the default `admin` route. The administrator path must not
overlap login, logout, service, terminal, signup, or reserved asset paths.
`config-check` prints the effective administrator URL. Rebuild and restart the
services to apply these changes; `init` includes the setting in new files and
leaves existing configuration files unchanged.

Create
accounts through registration or `add-user`; assigning an administrator ID does
not create the account automatically.

The interface shows users, account locks, session counts, container states, and
usage for mounted home disks. Actions include locking, unlocking, password resets,
disconnecting, restarting environments, and assigning classes or templates.
The dashboard includes summary counts, user search, state filters, and home
storage meters. Rows are updated in place, preserving search text and keyboard
focus. Manual and automatic refreshes share a single request. Automatic updates
run 15 seconds after the preceding request finishes; they pause while a dialog
is open, an action is running, or the tab is hidden. The Auto-refresh checkbox
allows manual control. Failed updates retain the last successful display and
show an error with its last update time. Disk service failures produce an explicit
warning while retaining available container and session status. The web interface rejects locking your own
administrator account to prevent accidental lockout. Password resets require
confirmation. Updating your own account invalidates the current session; the
interface stops polling and provides a sign-in link. Account changes that succeed
before a runtime error are explicitly reported, without automatic action retries.

Every administrator API request rechecks the account and administrator role.
Mutation requests require a session-bound CSRF token. Responses exclude account
hashes and secrets. Administrator and `healthz` paths are reserved. Manager's
operations API uses the existing shared secret; the host disk API is exposed only
through its permission-restricted Unix socket.
