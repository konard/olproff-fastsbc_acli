# Operations Manual

## Configuration

Environment variables:

- `FASTSBC_ACLI_LISTEN`: SSH listen address. Default: `:22`.
- `FASTSBC_ACLI_BACKEND`: backend gRPC address reserved for generated client wiring. Default: `127.0.0.1:50051`.
- `FASTSBC_ACLI_BACKEND_TIMEOUT`: per-command backend timeout. Default: `5s`.
- `FASTSBC_ACLI_AUDIT_LOG`: JSONL audit log path. Default: `audit/fastsbc_acli.jsonl`.
- `FASTSBC_ACLI_AUDIT_QUEUE_SIZE`: async audit queue size. Default: `1024`.
- `FASTSBC_ACLI_HOST_KEY`: optional SSH private host key path. If unset, an ephemeral Ed25519 key is generated for the process.
- `FASTSBC_ACLI_PROMPT_HOSTNAME`: prompt hostname. Default: `fastsbc`.
- `FASTSBC_ACLI_USERS`: comma-separated users in `username:password:role` form.

## Audit Logging

Every mutating command records:

- timestamp;
- client IP;
- username;
- role;
- raw command string;
- execution status;
- gRPC trace ID.

The writer is asynchronous to avoid adding terminal latency. The daemon tracks dropped events when the queue is full.

## Troubleshooting

- `% Invalid input detected at '^' marker.`: the command cannot be resolved in the current context.
- `% Incomplete command`: the command branch exists but needs more tokens or arguments.
- `% Ambiguous command`: the abbreviation matches multiple visible commands.
- `% Backend Unavailable`: the backend call timed out or was canceled.

For local development, use port `2222` unless running with privileges needed to bind port `22`.
