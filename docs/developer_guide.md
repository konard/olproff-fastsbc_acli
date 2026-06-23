# Developer Guide

## Add A Command

1. Locate the correct ACLI hierarchy in the Oracle Communications Session Border Controller ACLI Reference Guide.
2. Add the branch to `NewRootCommandTree` or `NewConfigureCommandTree` in `internal/acli/command.go`.
3. Keep command and parameter names snake_case or vendor-exact where the ACLI reference requires hyphenated names.
4. Add `ArgSpec` validators for syntax checks such as IP addresses, ranges, names, or enums.
5. Keep business logic out of the CLI. Forward accepted commands to `BackendClient`.
6. Add tests for abbreviation, completion visibility, error formatting, RBAC, and backend request shape.

## Candidate Configuration

Configuration commands should stage changes into `Session.Candidate`. Only `done` or `commit` may call `CommitCandidate`. Use stable dotted keys such as `system.hostname` internally and map them to protobuf fields or backend-specific payloads at the boundary.

## RBAC

Each command node declares a minimum role:

- `read_only`
- `operator`
- `admin`

Commands above the user's role are hidden from execution and completion.

## Local Environment

```sh
export FASTSBC_ACLI_LISTEN=:2222
export FASTSBC_ACLI_USERS='admin:admin:admin,viewer:viewer:read_only'
go run ./cmd/fastsbc-acli
```

The daemon uses an in-process mock backend until the generated gRPC client is wired to the C++ core.
