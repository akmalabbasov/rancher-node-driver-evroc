package evrocdriver

import (
	"testing"

	"github.com/akmalabbasov/evroc-sdk/client"
)

func TestEvrocMachineConfigToMachineSpecDefaults(t *testing.T) {
	cfg := EvrocMachineConfig{
		NamePrefix:        "test-node",
		ProjectID:         "project-1",
		SSHAuthorizedKeys: []string{"ssh-ed25519 AAAA test"},
	}

	normalized, err := cfg.Normalize()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	spec, err := normalized.toMachineSpec()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if spec.Region != "se-sto" {
		t.Fatalf("expected default region se-sto, got %q", spec.Region)
	}
	if spec.Zone != "a" {
		t.Fatalf("expected default zone a, got %q", spec.Zone)
	}
	if spec.ComputeProfileRef != "/compute/global/computeProfiles/a1a.s" {
		t.Fatalf("unexpected compute profile %q", spec.ComputeProfileRef)
	}
	if spec.DiskImageRef != "/compute/global/diskImages/evroc/ubuntu.24-04.1" {
		t.Fatalf("unexpected disk image ref %q", spec.DiskImageRef)
	}
	if spec.DiskSizeGB != 30 {
		t.Fatalf("expected default disk size 30, got %d", spec.DiskSizeGB)
	}
}

func TestEvrocMachineConfigToMachineSpecRequiresSSHKeys(t *testing.T) {
	_, err := (EvrocMachineConfig{NamePrefix: "test", ProjectID: "project-1"}).Normalize()
	if err == nil {
		t.Fatal("expected error for missing SSH keys")
	}
}

func TestEvrocMachineConfigNormalizeTrimsAndValidates(t *testing.T) {
	cfg := EvrocMachineConfig{
		NamePrefix:        "  test-node  ",
		ProjectID:         "  project-1  ",
		DiskSizeGB:        -1,
		SSHAuthorizedKeys: []string{"   ", " ssh-ed25519 AAAA test "},
	}

	_, err := cfg.Normalize()
	if err == nil {
		t.Fatal("expected validation error for negative disk size")
	}

	cfg.DiskSizeGB = 20
	normalized, err := cfg.Normalize()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if normalized.NamePrefix != "test-node" {
		t.Fatalf("unexpected normalized name prefix %q", normalized.NamePrefix)
	}
	if normalized.ProjectID != "project-1" {
		t.Fatalf("unexpected normalized project ID %q", normalized.ProjectID)
	}
	if len(normalized.SSHAuthorizedKeys) != 1 || normalized.SSHAuthorizedKeys[0] != "ssh-ed25519 AAAA test" {
		t.Fatalf("unexpected normalized ssh keys %#v", normalized.SSHAuthorizedKeys)
	}
}

func TestEvrocMachineConfigValidateRejectsInvalidCIDR(t *testing.T) {
	_, err := (EvrocMachineConfig{
		NamePrefix:              "test-node",
		ProjectID:               "project-1",
		SSHAuthorizedKeys:       []string{"ssh-ed25519 AAAA test"},
		SSHSourceCIDR:           "not-a-cidr",
		KubernetesAPISourceCIDR: "0.0.0.0/0",
	}).Normalize()
	if err == nil {
		t.Fatal("expected error for invalid sshSourceCIDR")
	}
}

func TestEvrocMachineConfigValidateRejectsInvalidSSHKey(t *testing.T) {
	_, err := (EvrocMachineConfig{
		NamePrefix:              "test-node",
		ProjectID:               "project-1",
		SSHAuthorizedKeys:       []string{"not-an-ssh-key"},
		SSHSourceCIDR:           "0.0.0.0/0",
		KubernetesAPISourceCIDR: "0.0.0.0/0",
	}).Normalize()
	if err == nil {
		t.Fatal("expected error for invalid SSH public key")
	}
}

func TestDriverResultFromStatusRunning(t *testing.T) {
	status := &Status{
		ProjectID: "project-1",
		Region:    "se-sto",
		Zone:      "a",
		PublicIP: &client.PublicIP{
			Status: struct {
				PublicIPv4Address string `json:"publicIPv4Address,omitempty"`
				UsedByRef         string `json:"usedByRef,omitempty"`
			}{PublicIPv4Address: "203.0.113.10"},
		},
		VirtualMachine: &client.VirtualMachine{
			Metadata: client.Metadata{ID: "node-1"},
			Status: struct {
				Networking struct {
					PrivateIPv4Address string `json:"privateIPv4Address,omitempty"`
					PublicIPv4Address  string `json:"publicIPv4Address,omitempty"`
				} `json:"networking,omitempty"`
				VirtualMachineStatus string `json:"virtualMachineStatus,omitempty"`
			}{VirtualMachineStatus: "Running"},
		},
	}
	status.VirtualMachine.Status.Networking.PrivateIPv4Address = "10.0.0.5"

	result, err := driverResultFromStatus(status)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.State != DriverStateRunning {
		t.Fatalf("expected running state, got %q", result.State)
	}
	if result.SSHCommand != "ssh ubuntu@203.0.113.10" {
		t.Fatalf("unexpected ssh command %q", result.SSHCommand)
	}
	if result.PrivateIP != "10.0.0.5" {
		t.Fatalf("unexpected private IP %q", result.PrivateIP)
	}
}

func TestDriverResultFromStatusMissing(t *testing.T) {
	result, err := driverResultFromStatus(&Status{ProjectID: "project-1", Region: "se-sto", Zone: "a"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.State != DriverStateMissing {
		t.Fatalf("expected missing state, got %q", result.State)
	}
}
