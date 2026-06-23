# Contributing

## Source License Headers

When adding a new source file, run `ninja -C build update-license` to apply the GPLv3 header.

The license tool is idempotent and leaves files untouched when the header already matches. CI runs `scripts/update_license_header.py --dry-run` and fails when a supported source file is missing the project header.

## Local Checks

Run the following before opening or updating a pull request:

```sh
go test ./...
python3 -m unittest discover -s scripts -p '*_test.py'
meson setup build
ninja -C build check-license
ninja -C build go-test
```

When `protoc`, `protoc-gen-go`, and `protoc-gen-go-grpc` are installed, generate protobuf stubs with:

```sh
ninja -C build generate-proto
```
