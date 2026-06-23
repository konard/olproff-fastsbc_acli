# fastsbc_acli

`fastsbc_acli` is the Administration Command Line Interface service for the Session Border Controller (SBC). It accepts administrator SSH sessions, parses Oracle-style ACLI commands, validates syntax and RBAC visibility, stages candidate configuration changes per session, and dispatches accepted operations to the SBC backend contract.

## Current Scope

This repository contains the initial service scaffold for issue [olproff/fastsbc_acli#1](https://github.com/olproff/fastsbc_acli/issues/1):

- Go SSH daemon with mandatory PTY session handling.
- ACLI parser with command abbreviation, `?` help, dynamic prompts, Oracle-style error formatting, and context-specific completions.
- Per-session candidate configuration with explicit `done`/`commit` and `cancel`.
- Backend client interface plus a mock backend for local development and tests.
- Asynchronous JSONL audit logging with trace IDs.
- Snake-case protobuf contract and Meson targets for license, Go, and protobuf workflows.
- Architecture, API, developer, and operations documentation in `docs/`.

The full Oracle ACLI command dictionary is intentionally data-driven work for follow-up changes. New command branches should be added through the command tree and protobuf contract described in the developer guide.

## Run Locally

Set at least one login user before starting the daemon:

```sh
export FASTSBC_ACLI_LISTEN=:2222
export FASTSBC_ACLI_USERS='admin:admin:admin,viewer:viewer:read_only'
go run ./cmd/fastsbc-acli
```

Then connect with:

```sh
ssh -p 2222 admin@127.0.0.1
```

## Build And Test

```sh
go test ./...
python3 -m unittest discover -s scripts -p '*_test.py'
meson setup build
ninja -C build check-license
ninja -C build go-test
```
