# Architecture Design

## Roles

`fastsbc_acli` has two control-plane roles:

- SSH server: accepts administrator SSH clients, requires a PTY session, and owns terminal input/output.
- Backend client: translates validated ACLI commands into requests for the C++ SBC core through the protobuf/gRPC contract.

The current implementation ships a mock backend to make the parser, session, transaction, and audit paths testable before the C++ engine is available.

## Session Isolation

Each SSH channel creates a `Session` with:

- authenticated username and role;
- client IP;
- current command path;
- private `CandidateConfig`;
- backend and audit handles.

Candidate configuration is never shared between sessions. `done` and `commit` send the current candidate snapshot to the backend. `cancel` clears the candidate without touching the running configuration.

## Command Tree

Commands are represented as RBAC-aware tree nodes. The resolver accepts unambiguous prefixes, hides commands above the user's role from execution and completion, and returns Oracle-style terminal errors:

- `% Incomplete command`
- `% Ambiguous command`
- `% Invalid input detected at '^' marker.`
- `% Backend Unavailable`

The initial command tree covers the interaction primitives needed to extend the ACLI:

- `show configuration`
- `show health`
- `configure terminal`
- `system hostname <name>`
- `network-interface ip-address <address>`
- `done` / `commit`
- `cancel`
- `exit`

## Concurrency

The SSH listener accepts connections in independent goroutines. Candidate state and command context live inside each session. Shared components use explicit synchronization:

- `MockBackendClient` protects captured calls with a mutex.
- `FileAuditLogger` uses a bounded channel and writer goroutine.
- `CandidateConfig` protects staged changes with a mutex.

## Shutdown

The daemon listens for `SIGINT` and `SIGTERM`, closes the SSH listener, and waits for the accept loop to exit. Command handlers execute with a configurable context timeout so stalled backend calls cannot block a session indefinitely.
