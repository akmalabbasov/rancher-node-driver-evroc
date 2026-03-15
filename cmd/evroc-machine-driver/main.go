package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/akmalabbasov/evroc-sdk/client"
	"github.com/akmalabbasov/rancher-node-driver-evroc/internal/drivercli"
	evrocdriver "github.com/akmalabbasov/rancher-node-driver-evroc/pkg/evrocdriver"
	"github.com/akmalabbasov/rancher-node-driver-evroc/pkg/machinedriver"
)

const commandTimeout = 2 * time.Minute

func main() {
	if len(os.Args) < 2 {
		fatalf("usage: %s <create|remove|state|ssh-host|ssh-user|ssh-command|result|schema|example-config> -config <path> -machine-config <path>", os.Args[0])
	}

	cmd := os.Args[1]
	fs := flag.NewFlagSet(cmd, flag.ExitOnError)
	configPath := fs.String("config", "./config.yaml", "Path to Evroc auth config")
	machineConfigPath := fs.String("machine-config", "./machine-config.example.yaml", "Path to machine config")
	fs.Parse(os.Args[2:])

	if cmd == "schema" {
		if err := evrocdriver.PrintJSON(machinedriver.Schema()); err != nil {
			fatalf("print schema failed: %v", err)
		}
		return
	}
	if cmd == "example-config" {
		fmt.Print(machinedriver.ExampleYAML())
		return
	}

	cfg, err := drivercli.LoadConfig(*configPath)
	if err != nil {
		fatalf("load config: %v", err)
	}
	machineCfg, err := drivercli.LoadMachineConfig(*machineConfigPath)
	if err != nil {
		fatalf("load machine config: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()

	clientFactory := func(ctx context.Context) (*client.Client, error) {
		return cfg.NewClient(ctx)
	}
	nd, err := evrocdriver.NewNodeDriver(clientFactory, *machineCfg)
	if err != nil {
		fatalf("build node driver: %v", err)
	}
	runtime := machinedriver.New(nd)

	switch cmd {
	case "create":
		if err := runtime.Create(ctx); err != nil {
			fatalf("create failed: %v", err)
		}
		fmt.Println("created")
	case "remove":
		if err := runtime.Remove(ctx); err != nil {
			fatalf("remove failed: %v", err)
		}
		fmt.Println("removed")
	case "state":
		state, err := runtime.GetState(ctx)
		if err != nil {
			fatalf("state failed: %v", err)
		}
		fmt.Println(state)
	case "ssh-host":
		host, err := runtime.GetSSHHostname(ctx)
		if err != nil {
			fatalf("ssh-host failed: %v", err)
		}
		fmt.Println(host)
	case "ssh-user":
		fmt.Println(runtime.GetSSHUsername())
	case "ssh-command":
		cmd, err := runtime.GetSSHCommand(ctx)
		if err != nil {
			fatalf("ssh-command failed: %v", err)
		}
		fmt.Println(cmd)
	case "result":
		result, err := runtime.GetResult(ctx)
		if err != nil {
			fatalf("result failed: %v", err)
		}
		if err := evrocdriver.PrintJSON(result); err != nil {
			fatalf("print result failed: %v", err)
		}
	default:
		fatalf("unknown command %q", cmd)
	}
}

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
