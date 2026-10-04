from pathlib import Path
import os
import subprocess
import tempfile
import unittest
from unittest.mock import patch

import migrate_system_paths as migration


class MigrationTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="linuxus migration ")
        self.addCleanup(self.temp.cleanup)
        base = Path(self.temp.name)
        self.source = base / "old deployment"
        self.source.mkdir()
        self.state = base / "state"
        self.config = base / "etc" / "linuxus" / ".env"
        self.ctl = self.source / "linuxusctl"
        self.ctl.touch()
        for name in ("data", "volumes"):
            (self.source / name).mkdir()
        (self.source / "data" / "AUTH_LIST").write_text("alice:unchanged\n")
        self.image = self.source / "volumes" / "share.img"
        with self.image.open("wb") as disk:
            disk.write(b"existing filesystem")
            disk.truncate(8 * 1024 * 1024)
        self.before = self.image.stat()
        self.original_config = "# preserve settings\nAUTH_SERVICE_SECURITY_SESSION_SECRET='private value'\n"
        for key, relative in migration.STORAGE.items():
            self.original_config += f"{key}='{relative}'\n"
        (self.source / ".env").write_text(self.original_config)
        self.commands = []
        self.mounts = [self.source / "volumes" / "share"]
        self.loops = ["/dev/fake-loop"]

    def run_command(self, args, **kwargs):
        self.commands.append(args)
        if str(args[0]) == str(self.ctl):
            self.assertEqual(kwargs["env"]["LINUXUS_CONFIG"], str(self.source / ".env"))
        elif args[0] == "umount":
            self.mounts.remove(args[-1])
        elif args[0] == "losetup":
            self.loops.remove(args[-1])
        return subprocess.CompletedProcess(args, 0)

    def perform(self):
        with patch.object(migration, "mounted_paths", side_effect=lambda: list(self.mounts)), patch.object(migration, "volume_loops", side_effect=lambda root: list(self.loops)), patch.object(migration, "run", side_effect=self.run_command):
            migration.migrate(self.source, self.ctl, self.config, self.state)

    def test_move_preserves_images_settings_and_legacy_links(self):
        self.perform()
        after = (self.state / "volumes" / "share.img").stat()
        with (self.state / "volumes" / "share.img").open("rb") as image:
            self.assertEqual(image.read(len(b"existing filesystem")), b"existing filesystem")
        self.assertEqual((self.before.st_ino, self.before.st_size, self.before.st_blocks, self.before.st_uid, self.before.st_gid, self.before.st_mode), (after.st_ino, after.st_size, after.st_blocks, after.st_uid, after.st_gid, after.st_mode))
        self.assertEqual((self.source / "data" / "AUTH_LIST").read_text(), "alice:unchanged\n")
        self.assertTrue((self.source / "data").is_symlink())
        self.assertTrue((self.source / "volumes").is_symlink())
        self.assertIn("AUTH_SERVICE_SECURITY_SESSION_SECRET='private value'", self.config.read_text())
        self.assertEqual(self.config.stat().st_mode & 0o777, 0o600)
        self.assertEqual((self.source / ".env").read_text(), self.original_config)
        self.assertEqual([str(cmd[0]) for cmd in self.commands], [str(self.ctl), "umount", "losetup"])

    def test_existing_destination_rejected_before_stopping_services(self):
        self.state.mkdir()
        with self.assertRaisesRegex(RuntimeError, "already exists"):
            self.perform()
        self.assertEqual(self.commands, [])
        self.assertTrue(self.image.is_file())

    def test_unmount_failure_leaves_data_at_old_paths(self):
        original = self.run_command
        def fail(args, **kwargs):
            if args[0] == "umount":
                raise subprocess.CalledProcessError(1, args)
            return original(args, **kwargs)
        self.run_command = fail
        with self.assertRaises(subprocess.CalledProcessError):
            self.perform()
        self.assertTrue(self.image.is_file())
        self.assertFalse(self.state.exists())
        self.assertFalse(self.config.exists())

    def test_publish_failure_rolls_back_renames(self):
        with patch.object(migration.os, "link", side_effect=PermissionError("publish failed")):
            with self.assertRaises(PermissionError):
                self.perform()
        self.assertFalse((self.source / "data").is_symlink())
        self.assertTrue(self.image.is_file())
        self.assertEqual(self.image.stat().st_ino, self.before.st_ino)
        self.assertFalse(self.state.exists())
        self.assertFalse(self.config.exists())

    def test_interrupt_after_publication_restores_original_layout(self):
        real_link = os.link
        def interrupt_after_link(source, target):
            real_link(source, target)
            raise KeyboardInterrupt()
        with patch.object(migration.os, "link", side_effect=interrupt_after_link):
            with self.assertRaises(KeyboardInterrupt):
                self.perform()
        self.assertFalse(self.config.exists())
        self.assertFalse(self.state.exists())
        self.assertTrue(self.image.is_file())
        self.assertFalse((self.source / "data").is_symlink())

    def test_custom_paths_rejected_before_stopping_services(self):
        (self.source / ".env").write_text(self.original_config.replace("'volumes/homes'", "'/other/homes'"))
        with self.assertRaisesRegex(RuntimeError, "Custom storage"):
            self.perform()
        self.assertEqual(self.commands, [])


if __name__ == "__main__":
    unittest.main()
