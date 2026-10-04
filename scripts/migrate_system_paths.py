#!/usr/bin/env python3
"""Move a standard legacy deployment to system paths without copying disk images.

The old data/ and volumes/ directories become compatibility symlinks. This
one-time helper requires root and a single filesystem for directory renames.
"""
import argparse
from contextlib import ExitStack
import fcntl
import json
import os
from pathlib import Path
import re
import shlex
import subprocess
import tempfile


STORAGE = {
    "AUTH_SERVICE_MOUNTS_HOST_AUTH_LIST_PATH": "data/AUTH_LIST",
    "VOLUMES_HOST_VOLUMES": "volumes",
    "VOLUMES_HOST_HOMES": "volumes/homes",
    "VOLUMES_HOST_SHARE": "volumes/share",
    "VOLUMES_HOST_READONLY": "volumes/readonly",
}


def rewrite_config(text, source, destination):
    """Preserve non-storage settings verbatim; refuse custom legacy layouts."""
    seen = set()
    lines = []
    for line in text.splitlines(keepends=True):
        match = re.fullmatch(r"\s*(?:export\s+)?([A-Z_][A-Z_0-9]*)\s*=(.*)", line.rstrip("\r\n"))
        if match and match[1] in STORAGE:
            key, value = match.groups()
            if key in seen:
                raise RuntimeError(f"Duplicate storage setting: {key}")
            tokens = shlex.split(value, comments=True, posix=True)
            if len(tokens) != 1:
                raise RuntimeError(f"Unsupported storage value: {key}")
            path = Path(tokens[0])
            if not path.is_absolute():
                path = source / path
            if path.resolve() != (source / STORAGE[key]).resolve():
                raise RuntimeError(f"Custom storage path in {key}; use the manual migration procedure")
            seen.add(key)
            lines.append(f"{key}={shlex.quote(str(destination / STORAGE[key]))}\n")
        else:
            lines.append(line)
    if seen != STORAGE.keys():
        raise RuntimeError("Legacy config must specify all five host storage paths")
    return "".join(lines)


def run(args, **kwargs):
    return subprocess.run([str(arg) for arg in args], check=True, **kwargs)


def mounted_paths():
    unescape = lambda text: re.sub(r"\\(040|011|012|134)", lambda m: chr(int(m[1], 8)), text)
    result = []
    for line in Path("/proc/self/mountinfo").read_text().splitlines():
        fields = line.split()
        if len(fields) < 10 or " - " not in line:
            raise RuntimeError("Invalid kernel mount table")
        result.append(Path(unescape(fields[4])))
    return result


def within(path, root):
    return path == root or root in path.parents


def volume_loops(root):
    result = run(["losetup", "--json", "--output", "NAME,BACK-FILE"], capture_output=True, text=True)
    devices = []
    for device in json.loads(result.stdout).get("loopdevices", []):
        if device.get("back-file") and within(Path(device["back-file"]).resolve(), root):
            devices.append(device["name"])
    return devices


def migrate(source, ctl, config_file=Path("/etc/linuxus/.env"), state=Path("/var/lib/linuxus")):
    source, ctl = source.resolve(strict=True), ctl.resolve(strict=True)
    source_config = source / ".env"
    if source_config.is_symlink():
        raise RuntimeError("Use the original deployment directory, not a linked .env")
    if os.path.lexists(config_file) or os.path.lexists(state):
        raise RuntimeError("System configuration/storage already exists; refusing to overwrite it")
    roots = [source / name for name in ("data", "volumes")]
    device = state.parent.stat().st_dev
    for root in roots:
        if root.is_symlink() or not root.is_dir():
            raise RuntimeError(f"Expected an ordinary legacy directory: {root}")
        if root.stat().st_dev != device:
            raise RuntimeError("Source and /var/lib/linuxus must share a filesystem; use the manual copy procedure")
    mounts = mounted_paths()
    if any(root in mounts for root in roots) or any(within(p, roots[0]) for p in mounts):
        raise RuntimeError("Storage root/data contains a mount; use the manual migration procedure")
    updated = rewrite_config(source_config.read_text(), source, state)
    config_file.parent.mkdir(parents=True, mode=0o700, exist_ok=True)
    fd, pending_name = tempfile.mkstemp(prefix=".env-migration-", dir=config_file.parent)
    pending = Path(pending_name)
    try:
        with os.fdopen(fd, "w") as output:
            output.write(updated)
            output.flush()
            os.fsync(output.fileno())
        env = os.environ.copy()
        env["LINUXUS_CONFIG"] = str(source_config)
        run([ctl, "down"], env=env)
        with ExitStack() as locks:
            lock_dir = roots[0] / ".disk-service"
            lock_dir.mkdir(mode=0o700, exist_ok=True)
            for name in ("service.lock", "ensure.lock"):
                lock = locks.enter_context((lock_dir / name).open("a+"))
                fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
            volumes = roots[1]
            for mount in sorted((p for p in mounted_paths() if within(p, volumes)), key=lambda p: len(p.parts), reverse=True):
                run(["umount", "--", mount])
            if any(within(p, volumes) for p in mounted_paths()):
                raise RuntimeError("A volume remains mounted; storage was not moved")
            for device in volume_loops(volumes):
                run(["losetup", "--detach", device])
            if volume_loops(volumes):
                raise RuntimeError("A volume image remains attached; storage was not moved")
            state.mkdir(mode=0o700)
            moved, aliases = [], []
            try:
                for original in roots:
                    target = state / original.name
                    original.rename(target)
                    moved.append((original, target))
                for original, target in moved:
                    original.symlink_to(target, target_is_directory=True)
                    aliases.append(original)
                # Publish the complete private configuration without overwriting
                # a file created by another administrator during migration.
                os.link(pending, config_file)
            except BaseException as failure:
                recovery = []
                # An interrupt may arrive immediately after the link succeeds.
                # Remove only the configuration inode published by this helper.
                try:
                    if config_file.exists() and os.path.samefile(pending, config_file):
                        config_file.unlink()
                except OSError as err:
                    recovery.append(str(err))
                for alias in reversed(aliases):
                    try:
                        alias.unlink()
                    except OSError as err:
                        recovery.append(str(err))
                for original, target in reversed(moved):
                    try:
                        target.rename(original)
                    except OSError as err:
                        recovery.append(str(err))
                try:
                    state.rmdir()
                except OSError as err:
                    recovery.append(str(err))
                if recovery:
                    raise RuntimeError(f"Migration failed; recovery also failed: {'; '.join(recovery)}. Inspect {state} before restarting.") from failure
                raise
    finally:
        pending.unlink(missing_ok=True)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--from", dest="source", type=Path, default=Path(__file__).resolve().parent.parent, help="legacy deployment directory (default: repository root)")
    parser.add_argument("--ctl", type=Path, help="new linuxusctl binary (default: <legacy directory>/linuxusctl)")
    args = parser.parse_args()
    if os.geteuid() != 0:
        parser.exit(1, "Run this migration helper with sudo.\n")
    try:
        migrate(args.source, args.ctl or args.source / "linuxusctl")
    except (OSError, RuntimeError, ValueError, subprocess.CalledProcessError) as err:
        parser.exit(1, f"Migration stopped: {err}\nServices may be stopped; check the reported condition before restarting.\n")
    print("Migrated configuration to /etc/linuxus/.env and storage to /var/lib/linuxus.")
    print("Existing data/ and volumes/ links point to the new locations; disk image contents and ownership are preserved.")
    print("Start services with: sudo <path-to-linuxusctl> up")


if __name__ == "__main__":
    main()
