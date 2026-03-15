# Rancher Node Driver for Evroc

This repository contains the current Evroc machine lifecycle implementation for Rancher-style node provisioning on Evroc.

It is a standalone Go module and release target. It now includes a real Docker Machine plugin binary that Rancher can import as a custom node driver:

- Evroc-backed machine provisioning
- a real `docker-machine-driver-evroc` plugin binary
- Rancher-shaped machine configuration and auxiliary CLIs
- machine-driver-style runtime accessors
- schema and example generation for a future Rancher UI layer

## Current Status

What works today:

- create, inspect, and destroy a single Evroc VM with supporting resources
- expose `docker-machine-driver-evroc` for Rancher custom node driver import
- drive that lifecycle through a Rancher-shaped `NodeDriver` facade
- expose machine-driver-style commands for state and SSH access
- generate machine config schema and example YAML
- build release binaries in GitHub Actions on version tags

What does not exist yet:

- a Rancher UI extension
- multi-node or HA cluster orchestration inside this repo
- Evroc VM power actions wired to `start`, `stop`, `restart`, and `kill`

## Repository Layout

- `cmd/rancher-node-driver/`: standalone provisioner and Rancher-shaped driver CLI
- `cmd/evroc-machine-driver/`: machine-driver-style runtime shim CLI
- `cmd/docker-machine-driver-evroc/`: Rancher-importable Docker Machine plugin binary
- `internal/drivercli/`: config loading and Evroc client construction
- `pkg/evrocdriver/`: Evroc machine config, validation, provisioning, and driver facade
- `pkg/machinedriver/`: runtime wrapper, schema generation, example generation, and e2e test
- `pkg/rancherdriver/`: actual Docker Machine driver implementation for Evroc
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

There are two auth modes in this repository.

`docker-machine-driver-evroc`
- accepts an Evroc API bearer token directly via `--evroc-access-token`
- does not depend on local `~/.evroc/config.yaml`
- this is the binary intended for Rancher import

Auxiliary CLIs in `cmd/rancher-node-driver` and `cmd/evroc-machine-driver`
- read Evroc authentication from the Evroc CLI config file
- build an API client from the selected profile

For the auxiliary CLIs, the defaults are:

- config path: `~/.evroc/config.yaml`
- profile: `default`

You can override the path in `config.yaml` with `configPath`, or override the profile with `profile`.

## Configuration Files

There are two YAML inputs used by the auxiliary CLIs.

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

### `cmd/docker-machine-driver-evroc`

This is the binary Rancher should import as a custom node driver.

Driver name:

- `evroc`

Expected binary name:

- `docker-machine-driver-evroc`

Current create flags exposed to Rancher:

- `evroc-access-token`
- `evroc-project-id`
- `evroc-api-url`
- `evroc-region`
- `evroc-zone`
- `evroc-compute-profile-ref`
- `evroc-disk-image-ref`
- `evroc-disk-size-gb`
- `evroc-ssh-source-cidr`
- `evroc-kubernetes-api-source-cidr`
- `evroc-ssh-user`
- `evroc-ssh-port`
- `evroc-engine-port`

How it works:

- Rancher generates an SSH key for the machine
- the driver injects the generated public key into the Evroc VM
- the driver creates a security group, public IP, boot disk, and VM
- Rancher then connects over SSH and continues machine provisioning

Important limitation:

- the current generic flag surface does not hide the access token in Rancher UI
- a proper Rancher UI extension is still needed for cloud credentials and a better machine config UX

### Import Into Rancher

At a minimum, the release asset must expose the Linux driver binary built from:

- `cmd/docker-machine-driver-evroc`

High-level import flow:

1. open `Cluster Management -> Drivers -> Node Drivers`
2. add a custom node driver
3. use driver name `evroc`
4. point Rancher at the release asset URL for `docker-machine-driver-evroc`
5. activate the driver
6. create a node template or machine pool using the `evroc-*` fields above

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

- `docker-machine-driver-evroc_linux_amd64.tar.gz`
- `docker-machine-driver-evroc_linux_arm64.tar.gz`
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

- the repo now ships a Rancher-importable plugin binary, but the overall project is still in PoC scope
- the current scope is single-machine lifecycle validation
- the code currently assumes Ubuntu images compatible with the shipped cloud-init
- the release workflow only builds Linux binaries
- the imported Rancher driver currently uses a direct access token field rather than a dedicated cloud credential UI
- `start`, `stop`, `restart`, and `kill` are not implemented because the current Evroc SDK/client does not expose VM power operations

## Next Step

The next engineering steps are:

1. add a Rancher UI extension for Evroc cloud credentials and machine config
2. add Evroc VM power operations to the SDK and wire them into the machine driver
3. validate end-to-end cluster creation from Rancher against the released `docker-machine-driver-evroc` binary
