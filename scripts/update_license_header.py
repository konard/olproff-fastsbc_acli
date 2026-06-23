#!/usr/bin/env python3
"""Apply the fastsbc GPLv3 source header to comment-compatible files."""

from __future__ import annotations

import argparse
from dataclasses import dataclass
from pathlib import Path
import sys

PROJECT_NAME = "fastsbc_cli"
COPYRIGHT_YEAR = "2026"
COPYRIGHT_HOLDER = "fastsbc"

HEADER_LINES = [
    f"// {PROJECT_NAME} - Administration Command Line Interface (ACLI) Service for SBC.",
    f"// Copyright (C) {COPYRIGHT_YEAR} {COPYRIGHT_HOLDER}",
    "//",
    "// This program is free software: you can redistribute it and/or modify",
    "// it under the terms of the GNU General Public License as published by",
    "// the Free Software Foundation, either version 3 of the License, or",
    "// (at your option) any later version.",
    "//",
    "// This program is distributed in the hope that it will be useful,",
    "// but WITHOUT ANY WARRANTY; without even the implied warranty of",
    "// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the",
    "// GNU General Public License for more details.",
    "//",
    "// You should have received a copy of the GNU General Public License",
    "// along with this program. If not, see <https://www.gnu.org/licenses/>.",
]

SUPPORTED_EXTENSIONS = {
    ".c",
    ".cc",
    ".cpp",
    ".cxx",
    ".go",
    ".h",
    ".hh",
    ".hpp",
    ".hxx",
    ".proto",
}
EXCLUDED_DIRS = {
    ".git",
    ".github",
    "build",
    "dist",
    "gen",
    "third_party",
    "vendor",
}
GENERATED_SUFFIXES = (
    ".pb.go",
    "_grpc.pb.go",
    ".pb.cc",
    ".pb.h",
)


@dataclass
class Summary:
    processed: int = 0
    added: int = 0
    updated: int = 0
    skipped: int = 0

    @property
    def changed(self) -> int:
        return self.added + self.updated


def reference_header(newline: str) -> str:
    return newline.join(HEADER_LINES) + newline


def detect_newline(data: bytes) -> str:
    return "\r\n" if b"\r\n" in data else "\n"


def should_process(path: Path) -> bool:
    if path.suffix not in SUPPORTED_EXTENSIONS:
        return False
    if path.name.endswith(GENERATED_SUFFIXES):
        return False
    return not any(part in EXCLUDED_DIRS for part in path.parts)


def has_third_party_copyright(text: str) -> bool:
    probe = "\n".join(text.splitlines()[:20])
    if PROJECT_NAME in probe:
        return False
    if COPYRIGHT_HOLDER in probe:
        return False
    return "Copyright" in probe or "copyright" in probe


def split_existing_header(text: str) -> tuple[str | None, str]:
    lines = text.splitlines(keepends=True)
    if not lines:
        return None, ""
    if not lines[0].startswith(f"// {PROJECT_NAME} "):
        return None, text
    end_index = 0
    for index, line in enumerate(lines):
        if "https://www.gnu.org/licenses/" in line:
            end_index = index + 1
            break
    if end_index == 0:
        return None, text
    while end_index < len(lines) and lines[end_index].strip() == "":
        end_index += 1
    return "".join(lines[:end_index]).rstrip("\r\n"), "".join(lines[end_index:])


def normalize_after_header(rest: str, newline: str) -> str:
    return rest.lstrip("\r\n").lstrip("\n") if rest == "" else rest.lstrip("\r\n")


def process_file(path: Path, dry_run: bool) -> str:
    data = path.read_bytes()
    newline = detect_newline(data)
    text = data.decode("utf-8")
    header = reference_header(newline).rstrip("\r\n")

    if has_third_party_copyright(text):
        return "skipped"

    existing_header, rest = split_existing_header(text)
    if existing_header == header:
        return "processed"

    rest = normalize_after_header(rest, newline)
    new_text = reference_header(newline) + newline + rest
    if existing_header is None:
        status = "added"
    else:
        status = "updated"
    if not dry_run:
        path.write_bytes(new_text.encode("utf-8"))
    return status


def iter_source_files(root: Path):
    for path in root.rglob("*"):
        if path.is_file() and should_process(path.relative_to(root)):
            yield path


def process_tree(root: Path, dry_run: bool) -> Summary:
    summary = Summary()
    for path in iter_source_files(root):
        status = process_file(path, dry_run)
        if status == "processed":
            summary.processed += 1
        elif status == "added":
            summary.processed += 1
            summary.added += 1
            print(f"added header: {path}")
        elif status == "updated":
            summary.processed += 1
            summary.updated += 1
            print(f"updated header: {path}")
        elif status == "skipped":
            summary.skipped += 1
            print(f"skipped third-party copyright: {path}", file=sys.stderr)
    return summary


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", default=".", help="source tree root")
    parser.add_argument("--dry-run", action="store_true", help="report changes without writing files")
    args = parser.parse_args(argv)

    summary = process_tree(Path(args.root).resolve(), args.dry_run)
    print(
        "summary: "
        f"processed={summary.processed} "
        f"added={summary.added} "
        f"updated={summary.updated} "
        f"skipped={summary.skipped}"
    )
    if args.dry_run and summary.changed:
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
