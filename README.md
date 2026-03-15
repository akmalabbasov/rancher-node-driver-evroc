# Rancher Node Driver for Evroc

This repository contains the current Evroc machine lifecycle PoC for Rancher-style node provisioning.

It is a standalone Go module and release target. It is not yet a Rancher-installable node driver plugin, but it already implements the core pieces needed for one:

- Evroc-backed machine provisioning
- Rancher-shaped machine configuration
- machine-driver-style runtime accessors
- schema and example generation for a future Rancher UI layer

## Current Status

What works today:

- create, inspect, and destroy a single Evroc VM with supporting resources
- drive that lifecycle through a Rancher-shaped `NodeDriver` facade
- expose machine-driver-style commands for state and SSH access
- generate machine config schema and example YAML
- build release binaries in GitHub Actions on version tags

What does not exist yet:

- a real Rancher node-driver SDK/runtime integration
- a Rancher UI extension
- multi-node or HA cluster orchestration inside this repo

## Repository Layout

- `cmd/rancher-node-driver/`: standalone provisioner and Rancher-shaped driver CLI
- `cmd/evroc-machine-driver/`: machine-driver-style runtime shim CLI
- `internal/drivercli/`: config loading and Evroc client construction
- `pkg/evrocdriver/`: Evroc machine config, validation, provisioning, and driver facade
- `pkg/machinedriver/`: runtime wrapper, schema generation, example generation, and e2e test
- `config.example.yaml`: example CLI config for direct provisioning flow
- `machine-config.example.yaml`: example machine config for the driver/runtime flow
- `schema.json`: generated machine config schema artifact
- `.github/workflows/build.yml`: tagged release build for Linux binaries

## Requirements

- Go `1.24+`
- access to an Evroc account and project
- local Evroc CLI config at `~/.evroc/config.yaml`, or an explicit `configPath`

This repo uses the shared SDK module:

- `github.com/akmalabbasov/evroc-sdk`

## Authentication Model

The binaries do not accept raw API credentials directly.

They read Evroc authentication from the Evroc CLI config file and then build an API client from the selected profile. By default:

- config path: `~/.evroc/config.yaml`
- profile: `default`

You can override the path in `config.yaml` with `configPath`, or override the profile with `profile`.

## Configuration Files

There are two YAML inputs with different purposes.

`config.example.yaml`
- used by `cmd/rancher-node-driver`
- includes Evroc profile selection plus machine settings
- suitable for direct provisioner testing

`machine-config.example.yaml`
- used by `driver-*` commands and `cmd/evroc-machine-driver`
- contains only the Rancher-facing machine configuration surface

### Machine Config Fields

Current machine config fields:

- `namePrefix`
- `projectID`
- `region`
- `zone`
- `computeProfileRef`
- `diskImageRef`
- `diskSizeGB`
- `sshAuthorizedKeys`
- `sshSourceCIDR`
- `kubernetesAPISourceCIDR`

Current defaults:

- `region`: `se-sto`
- `zone`: `a`
- `computeProfileRef`: `/compute/global/computeProfiles/a1a.s`
- `diskImageRef`: `/compute/global/diskImages/evroc/ubuntu.24-04.1`
- `diskSizeGB`: `30`
- `sshSourceCIDR`: `0.0.0.0/0`
- `kubernetesAPISourceCIDR`: `0.0.0.0/0`

Validation currently enforces:

- `namePrefix` is required
- `projectID` is required
- at least one `sshAuthorizedKeys` entry is required
- `sshSourceCIDR` must be valid CIDR
- `kubernetesAPISourceCIDR` must be valid CIDR
- SSH keys must look like supported public key types

## Commands

### `cmd/rancher-node-driver`

This CLI supports:

- `create`
- `status`
- `destroy`
- `driver-create`
- `driver-state`
- `driver-remove`

Common flags:

- `-config`: path to config YAML, default `./config.yaml`
- `-machine-config`: path to machine config YAML, default `./machine-config.example.yaml`
- `-json`: structured output for `create` and driver commands

Create flow flags:

- `-wait`: wait for refreshed resource status after create
- `-wait-ssh`: wait for TCP reachability on port `22` after create

Examples:

```bash
go run ./cmd/rancher-node-driver create -wait -config ./config.yaml
go run ./cmd/rancher-node-driver create -wait-ssh -json -config ./config.yaml
go run ./cmd/rancher-node-driver status -config ./config.yaml
go run ./cmd/rancher-node-driver destroy -config ./config.yaml
```

Driver facade examples:

```bash
go run ./cmd/rancher-node-driver driver-create -json \
  -config ./config.yaml \
  -machine-config ./machine-config.example.yaml

go run ./cmd/rancher-node-driver driver-state \
  -config ./config.yaml \
  -machine-config ./machine-config.example.yaml

go run ./cmd/rancher-node-driver driver-remove \
  -config ./config.yaml \
  -machine-config ./machine-config.example.yaml
```

### `cmd/evroc-machine-driver`

This CLI exposes the runtime-style surface that a future Rancher integration would need.

Supported commands:

- `create`
- `remove`
- `state`
- `ssh-host`
- `ssh-user`
- `ssh-command`
- `result`
- `schema`
- `example-config`

Examples:

```bash
go run ./cmd/evroc-machine-driver result \
  -config ./config.yaml \
  -machine-config ./machine-config.example.yaml

go run ./cmd/evroc-machine-driver state \
  -config ./config.yaml \
  -machine-config ./machine-config.example.yaml

go run ./cmd/evroc-machine-driver schema
go run ./cmd/evroc-machine-driver example-config
```

## Generated Artifacts

Generated from the machine config schema:

- `schema.json`
- `machine-config.example.yaml`

Regenerate them with:

```bash
make generate
```

## Development

Run tests:

```bash
make test
```

`make test` uses:

- `GOWORK=off`
- `GOPRIVATE=github.com/akmalabbasov/evroc-sdk`
- `GONOSUMDB=github.com/akmalabbasov/evroc-sdk`

Those environment variables are also used in the release workflow so the repo behaves the same way locally and in CI.

## End-to-End Test

The e2e lifecycle test is opt-in because it creates and destroys real Evroc resources.

Run it with:

```bash
EVROC_E2E=1 \
EVROC_DRIVER_CONFIG_PATH=./config.yaml \
EVROC_MACHINE_CONFIG_PATH=./machine-config.example.yaml \
GOPRIVATE=github.com/akmalabbasov/evroc-sdk \
GONOSUMDB=github.com/akmalabbasov/evroc-sdk \
go test ./pkg/machinedriver -run TestE2EMachineLifecycle -count=1
```

## Release Process

GitHub Actions builds release binaries on tags matching `v*`.

Current outputs:

- `rancher-node-driver_linux_amd64.tar.gz`
- `rancher-node-driver_linux_arm64.tar.gz`
- `evroc-machine-driver_linux_amd64.tar.gz`
- `evroc-machine-driver_linux_arm64.tar.gz`

Workflow:

1. push `main`
2. create a tag such as `v0.1.1`
3. push the tag
4. GitHub Actions uploads artifacts and attaches them to the GitHub release

## Known Limitations

- the repo is still a PoC, not a Rancher-installable plugin
- the current scope is single-machine lifecycle validation
- the code currently assumes Ubuntu images compatible with the shipped cloud-init
- the release workflow only builds Linux binaries

## Next Step

The next engineering step is to replace the local runtime shim with Rancher's real machine-driver integration surface while reusing:

- `pkg/evrocdriver`
- `pkg/machinedriver`
- `schema.json`
