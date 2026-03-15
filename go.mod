module github.com/akmalabbasov/rancher-node-driver-evroc

go 1.25.0

require (
	github.com/akmalabbasov/evroc-sdk v0.1.0
	github.com/docker/machine v0.16.2
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/Azure/go-ansiterm v0.0.0-20250102033503-faa5f7b0171c // indirect
	github.com/docker/docker v20.10.24+incompatible // indirect
	github.com/moby/term v0.5.2 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	golang.org/x/crypto v0.49.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/term v0.41.0 // indirect
)

replace github.com/docker/machine => github.com/rancher/machine v0.16.2
