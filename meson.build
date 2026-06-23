project(
  'fastsbc_acli',
  ['c', 'cpp'],
  version: '0.1.0',
  license: 'GPL-3.0-or-later',
  meson_version: '>=1.0.0',
)

python = find_program('python3')
go = find_program('go', required: false)
protoc = find_program('protoc', required: false)

run_target(
  'update-license',
  command: [python, files('scripts/update_license_header.py'), '--root', meson.current_source_dir()],
)

run_target(
  'check-license',
  command: [python, files('scripts/update_license_header.py'), '--root', meson.current_source_dir(), '--dry-run'],
)

if go.found()
  run_target(
    'go-test',
    command: [go, '-C', meson.current_source_dir(), 'test', './...'],
  )
  run_target(
    'go-build',
    command: [go, '-C', meson.current_source_dir(), 'build', './cmd/fastsbc-acli'],
  )
endif

run_target(
  'test-license-tool',
  command: [python, '-m', 'unittest', 'discover', '-s', meson.current_source_dir() / 'scripts', '-p', '*_test.py'],
)

if protoc.found()
  gen_go_dir = meson.current_source_dir() / 'gen' / 'go'
  proto_root = meson.current_source_dir() / 'proto'
  proto_file = proto_root / 'acli' / 'v1' / 'acli.proto'
  run_target(
    'generate-proto',
    command: [
      protoc,
      '-I', proto_root,
      '--go_out=' + gen_go_dir,
      '--go_opt=paths=source_relative',
      '--go-grpc_out=' + gen_go_dir,
      '--go-grpc_opt=paths=source_relative',
      proto_file,
    ],
  )
else
  warning('protoc was not found; generate-proto target is unavailable')
endif
