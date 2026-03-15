package main

import (
	"github.com/akmalabbasov/rancher-node-driver-evroc/pkg/rancherdriver"
	"github.com/docker/machine/libmachine/drivers/plugin"
)

func main() {
	plugin.RegisterDriver(rancherdriver.NewDriver("", ""))
}
