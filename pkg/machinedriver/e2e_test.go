package machinedriver

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/akmalabbasov/evroc-sdk/client"
	"github.com/akmalabbasov/rancher-node-driver-evroc/internal/drivercli"
	evrocdriver "github.com/akmalabbasov/rancher-node-driver-evroc/pkg/evrocdriver"
)

func TestE2EMachineLifecycle(t *testing.T) {
	if os.Getenv("EVROC_E2E") != "1" {
		t.Skip("set EVROC_E2E=1 to run end-to-end Evroc lifecycle test")
	}

	configPath := os.Getenv("EVROC_DRIVER_CONFIG_PATH")
	machineConfigPath := os.Getenv("EVROC_MACHINE_CONFIG_PATH")
	if configPath == "" || machineConfigPath == "" {
		t.Fatal("EVROC_DRIVER_CONFIG_PATH and EVROC_MACHINE_CONFIG_PATH are required for EVROC_E2E=1")
	}

	cfg, err := drivercli.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	machineCfg, err := drivercli.LoadMachineConfig(machineConfigPath)
	if err != nil {
		t.Fatalf("load machine config: %v", err)
	}

	machineCfg.NamePrefix = machineCfg.NamePrefix + "-e2e"
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	clientFactory := func(ctx context.Context) (*client.Client, error) {
		return cfg.NewClient(ctx)
	}
	nodeDriver, err := evrocdriver.NewNodeDriver(clientFactory, *machineCfg)
	if err != nil {
		t.Fatalf("build node driver: %v", err)
	}
	runtime := New(nodeDriver)

	defer func() {
		_ = runtime.Remove(context.Background())
	}()

	if err := runtime.Create(ctx); err != nil {
		t.Fatalf("create: %v", err)
	}
	result, err := runtime.GetResult(ctx)
	if err != nil {
		t.Fatalf("get result: %v", err)
	}
	if result.State != StateRunning {
		t.Fatalf("expected running state, got %q", result.State)
	}
	if !strings.HasPrefix(result.NodeName, machineCfg.NamePrefix) {
		t.Fatalf("expected node name with prefix %q, got %q", machineCfg.NamePrefix, result.NodeName)
	}
	if result.PublicIP == "" {
		t.Fatal("expected public IP to be assigned")
	}
	if err := runtime.Remove(ctx); err != nil {
		t.Fatalf("remove: %v", err)
	}
}
