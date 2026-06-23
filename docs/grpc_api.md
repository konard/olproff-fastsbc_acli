# gRPC API Contract

The shared protobuf source of truth is `proto/acli/v1/acli.proto`. It uses snake_case names for services, RPC methods, messages, and fields to match the issue convention.

## Service

`acli_service` exposes:

- `execute_command`: forwards non-configuration operational commands.
- `commit_candidate`: applies a candidate configuration snapshot to the backend.
- `show_state`: reads live operational or configuration state.

Each request carries:

- `trace_id` for end-to-end correlation;
- `username`;
- `role`;
- `raw_command`;
- command arguments or candidate changes.

## Error Mapping

The CLI owns syntax and type validation. The backend owns business validation. Backend failures should be mapped at the CLI boundary as follows:

- `Unavailable` and `DeadlineExceeded`: `% Backend Unavailable`
- invalid resource relationships or uniqueness conflicts: `% <backend error message>`
- accepted requests: backend terminal output is passed through unchanged.

## Generation

Meson provides a `generate-proto` target when `protoc` is installed:

```sh
ninja -C build generate-proto
```

CI installs `protobuf-compiler`, `protoc-gen-go`, and `protoc-gen-go-grpc` before invoking this target.
