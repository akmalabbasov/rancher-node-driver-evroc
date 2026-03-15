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
)

const commandTimeout = 2 * time.Minute

func main() {
	if len(os.Args) < 2 {
		fatalf("usage: %s <create|status|destroy|driver-create|driver-state|driver-remove> -config <path>", os.Args[0])
	}

	cmd := os.Args[1]
	fs := flag.NewFlagSet(cmd, flag.ExitOnError)
	configPath := fs.String("config", "./config.yaml", "Path to PoC config file")
	machineConfigPath := fs.String("machine-config", "./machine-config.example.yaml", "Path to Rancher-style machine config file")
	wait := fs.Bool("wait", false, "Wait for refreshed status after create")
	waitSSH := fs.Bool("wait-ssh", false, "Wait for SSH reachability after create")
	jsonOutput := fs.Bool("json", false, "Print structured JSON output")
	fs.Parse(os.Args[2:])

	cfg, err := drivercli.LoadConfig(*configPath)
	if err != nil {
		fatalf("load config: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()

	clientFactory := func(ctx context.Context) (*client.Client, error) {
		return cfg.NewClient(ctx)
	}

	machineCfg := cfg.MachineConfig()
	if cmd == "driver-create" || cmd == "driver-state" || cmd == "driver-remove" {
		loaded, loadErr := drivercli.LoadMachineConfig(*machineConfigPath)
		if loadErr != nil {
			fatalf("load machine config: %v", loadErr)
		}
		machineCfg = *loaded
	}

	cli, err := cfg.NewClient(ctx)
	if err != nil {
		fatalf("create evroc client: %v", err)
	}
	prov, err := evrocdriver.NewProvisioner(cli, machineCfg)
	if err != nil {
		fatalf("build provisioner: %v", err)
	}
	driver, err := evrocdriver.NewNodeDriver(clientFactory, machineCfg)
	if err != nil {
		fatalf("build node driver: %v", err)
	}

	switch cmd {
	case "create":
		err = runCreate(ctx, prov, *wait, *waitSSH, *jsonOutput)
	case "status":
		err = runStatus(ctx, prov)
	case "destroy":
		err = runDestroy(ctx, prov)
	case "driver-create":
		err = runDriverCreate(ctx, driver, *jsonOutput)
	case "driver-state":
		err = runDriverState(ctx, driver)
	case "driver-remove":
		err = runDriverRemove(ctx, driver)
	default:
		fatalf("unknown command %q", cmd)
	}
	if err != nil {
		fatalf("%s failed: %v", cmd, err)
	}
}

func runCreate(ctx context.Context, prov *evrocdriver.Provisioner, wait, waitSSH, jsonOutput bool) error {
	status, err := prov.Create(ctx)
	if err != nil {
		return err
	}
	if wait || waitSSH {
		status, err = prov.WaitForStatus(ctx, evrocdriver.DefaultWaitTimeout)
		if err != nil {
			return err
		}
	}
	if waitSSH {
		if err := prov.WaitForSSH(ctx, evrocdriver.DefaultSSHWaitTimeout); err != nil {
			return err
		}
		status, err = prov.Status(ctx)
		if err != nil {
			return err
		}
	}

	summary, err := prov.Summary(status)
	if err != nil {
		return err
	}
	if jsonOutput {
		return evrocdriver.PrintJSON(summary)
	}

	fmt.Printf("Created or reused Rancher PoC node %s\n", summary.NodeName)
	fmt.Printf("Public IP: %s\n", summary.PublicIP)
	if summary.SSHCommand != "" {
		fmt.Printf("SSH: %s\n", summary.SSHCommand)
	} else {
		fmt.Println("SSH: pending public IP assignment")
	}
	fmt.Printf("VM state: %s\n", summary.VMState)
	fmt.Printf("Project/region/zone: %s/%s/%s\n", summary.ProjectID, summary.Region, summary.Zone)
	return nil
}

func runStatus(ctx context.Context, prov *evrocdriver.Provisioner) error {
	status, err := prov.Status(ctx)
	if err != nil {
		return err
	}
	return evrocdriver.PrintJSON(status)
}

func runDestroy(ctx context.Context, prov *evrocdriver.Provisioner) error {
	if err := prov.Destroy(ctx); err != nil {
		return err
	}
	fmt.Println("Destroyed Rancher PoC resources")
	return nil
}

func runDriverCreate(ctx context.Context, driver *evrocdriver.NodeDriver, jsonOutput bool) error {
	result, err := driver.Create(ctx)
	if err != nil {
		return err
	}
	if jsonOutput {
		return evrocdriver.PrintJSON(result)
	}
	fmt.Printf("Driver created node %s\n", result.NodeName)
	fmt.Printf("State: %s\n", result.State)
	fmt.Printf("Public IP: %s\n", result.PublicIP)
	fmt.Printf("SSH: %s\n", result.SSHCommand)
	return nil
}

func runDriverState(ctx context.Context, driver *evrocdriver.NodeDriver) error {
	result, err := driver.GetState(ctx)
	if err != nil {
		return err
	}
	return evrocdriver.PrintJSON(result)
}

func runDriverRemove(ctx context.Context, driver *evrocdriver.NodeDriver) error {
	if err := driver.Remove(ctx); err != nil {
		return err
	}
	fmt.Println("Driver removed node resources")
	return nil
}

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
