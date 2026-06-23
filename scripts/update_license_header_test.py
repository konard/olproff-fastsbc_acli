#!/usr/bin/env python3
"""Tests for update_license_header.py."""

from __future__ import annotations

import importlib.util
from pathlib import Path
import sys
import tempfile
import unittest


SCRIPT = Path(__file__).with_name("update_license_header.py")
SPEC = importlib.util.spec_from_file_location("update_license_header", SCRIPT)
update_license_header = importlib.util.module_from_spec(SPEC)
assert SPEC.loader is not None
sys.modules["update_license_header"] = update_license_header
SPEC.loader.exec_module(update_license_header)


class UpdateLicenseHeaderTest(unittest.TestCase):
    def test_inserts_header_before_go_build_tag(self):
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "main.go"
            path.write_text("//go:build linux\n\npackage main\n", encoding="utf-8")

            status = update_license_header.process_file(path, dry_run=False)

            self.assertEqual(status, "added")
            text = path.read_text(encoding="utf-8")
            self.assertTrue(text.startswith("// fastsbc_cli - Administration"))
            self.assertIn("\n\n//go:build linux\n", text)

    def test_is_idempotent_when_header_matches(self):
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "main.go"
            header = update_license_header.reference_header("\n")
            path.write_text(header + "\npackage main\n", encoding="utf-8")

            status = update_license_header.process_file(path, dry_run=False)

            self.assertEqual(status, "processed")
            self.assertEqual(path.read_text(encoding="utf-8"), header + "\npackage main\n")

    def test_skips_third_party_copyright(self):
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "vendor.go"
            path.write_text("// Copyright 2020 Other Corp\npackage vendor\n", encoding="utf-8")

            status = update_license_header.process_file(path, dry_run=False)

            self.assertEqual(status, "skipped")

    def test_dry_run_reports_pending_change(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            (root / "main.go").write_text("package main\n", encoding="utf-8")

            summary = update_license_header.process_tree(root, dry_run=True)

            self.assertEqual(summary.added, 1)
            self.assertEqual((root / "main.go").read_text(encoding="utf-8"), "package main\n")


if __name__ == "__main__":
    unittest.main()
