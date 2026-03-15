package evrocdriver

import (
	"context"
	"fmt"
	"net/netip"
	"strings"

	"github.com/akmalabbasov/evroc-sdk/client"
)

// EvrocMachineConfig is the Rancher-facing machine configuration shape for this PoC.
// It is intentionally close to what a future node-driver config would need.
type EvrocMachineConfig struct {
	NamePrefix              string   `json:"namePrefix" yaml:"namePrefix"`
	ProjectID               string   `json:"projectID" yaml:"projectID"`
	Region                  string   `json:"region" yaml:"region"`
	Zone                    string   `json:"zone" yaml:"zone"`
	ComputeProfileRef       string   `json:"computeProfileRef" yaml:"computeProfileRef"`
	DiskImageRef            string   `json:"diskImageRef" yaml:"diskImageRef"`
	DiskSizeGB              int32    `json:"diskSizeGB" yaml:"diskSizeGB"`
	SSHAuthorizedKeys       []string `json:"sshAuthorizedKeys" yaml:"sshAuthorizedKeys"`
	SSHSourceCIDR           string   `json:"sshSourceCIDR" yaml:"sshSourceCIDR"`
	KubernetesAPISourceCIDR string   `json:"kubernetesAPISourceCIDR" yaml:"kubernetesAPISourceCIDR"`
}

type DriverState string

const (
	DriverStateMissing      DriverState = "missing"
	DriverStateProvisioning DriverState = "provisioning"
	DriverStateRunning      DriverState = "running"
)

type NodeDriver struct {
	provisionerFactory func(context.Context) (*Provisioner, error)
}

type DriverResult struct {
	NodeName   string      `json:"nodeName"`
	State      DriverState `json:"state"`
	PublicIP   string      `json:"publicIP,omitempty"`
	PrivateIP  string      `json:"privateIP,omitempty"`
	SSHCommand string      `json:"sshCommand,omitempty"`
	VMState    string      `json:"vmState,omitempty"`
	ProjectID  string      `json:"projectID,omitempty"`
	Region     string      `json:"region,omitempty"`
	Zone       string      `json:"zone,omitempty"`
}

func NewNodeDriver(clientFactory func(context.Context) (*client.Client, error), machineCfg EvrocMachineConfig) (*NodeDriver, error) {
	normalized, err := machineCfg.Normalize()
	if err != nil {
		return nil, err
	}
	machineSpec, err := normalized.toMachineSpec()
	if err != nil {
		return nil, err
	}

	return &NodeDriver{
		provisionerFactory: func(ctx context.Context) (*Provisioner, error) {
			cli, err := clientFactory(ctx)
			if err != nil {
				return nil, err
			}
			return NewProvisionerFromSpec(cli, machineSpec), nil
		},
	}, nil
}

func (c EvrocMachineConfig) Normalize() (EvrocMachineConfig, error) {
	normalized := c
	normalized.NamePrefix = strings.TrimSpace(normalized.NamePrefix)
	normalized.ProjectID = strings.TrimSpace(normalized.ProjectID)
	normalized.Region = strings.TrimSpace(normalized.Region)
	normalized.Zone = strings.TrimSpace(normalized.Zone)
	normalized.ComputeProfileRef = strings.TrimSpace(normalized.ComputeProfileRef)
	normalized.DiskImageRef = strings.TrimSpace(normalized.DiskImageRef)
	normalized.SSHSourceCIDR = strings.TrimSpace(normalized.SSHSourceCIDR)
	normalized.KubernetesAPISourceCIDR = strings.TrimSpace(normalized.KubernetesAPISourceCIDR)

	if normalized.Region == "" {
		normalized.Region = "se-sto"
	}
	if normalized.Zone == "" {
		normalized.Zone = "a"
	}
	if normalized.ComputeProfileRef == "" {
		normalized.ComputeProfileRef = "/compute/global/computeProfiles/a1a.s"
	}
	if normalized.DiskImageRef == "" {
		normalized.DiskImageRef = "/compute/global/diskImages/evroc/ubuntu.24-04.1"
	}
	if normalized.DiskSizeGB == 0 {
		normalized.DiskSizeGB = 30
	}
	if normalized.SSHSourceCIDR == "" {
		normalized.SSHSourceCIDR = "0.0.0.0/0"
	}
	if normalized.KubernetesAPISourceCIDR == "" {
		normalized.KubernetesAPISourceCIDR = "0.0.0.0/0"
	}

	keys := make([]string, 0, len(normalized.SSHAuthorizedKeys))
	for _, key := range normalized.SSHAuthorizedKeys {
		trimmed := strings.TrimSpace(key)
		if trimmed != "" {
			keys = append(keys, trimmed)
		}
	}
	normalized.SSHAuthorizedKeys = keys

	if err := normalized.Validate(); err != nil {
		return EvrocMachineConfig{}, err
	}
	return normalized, nil
}

func (c EvrocMachineConfig) Validate() error {
	if c.NamePrefix == "" {
		return fmt.Errorf("namePrefix is required")
	}
	if c.ProjectID == "" {
		return fmt.Errorf("projectID is required")
	}
	if c.Region == "" {
		return fmt.Errorf("region must not be empty")
	}
	if c.Zone == "" {
		return fmt.Errorf("zone must not be empty")
	}
	if c.DiskSizeGB < 0 {
		return fmt.Errorf("diskSizeGB must be positive")
	}
	if len(c.SSHAuthorizedKeys) == 0 {
		return fmt.Errorf("sshAuthorizedKeys must contain at least one key")
	}
	if _, err := netip.ParsePrefix(c.SSHSourceCIDR); err != nil {
		return fmt.Errorf("sshSourceCIDR must be a valid CIDR: %w", err)
	}
	if _, err := netip.ParsePrefix(c.KubernetesAPISourceCIDR); err != nil {
		return fmt.Errorf("kubernetesAPISourceCIDR must be a valid CIDR: %w", err)
	}
	for _, key := range c.SSHAuthorizedKeys {
		if !looksLikeSSHPublicKey(key) {
			return fmt.Errorf("sshAuthorizedKeys contains an invalid SSH public key")
		}
	}
	return nil
}

func (d *NodeDriver) Create(ctx context.Context) (*DriverResult, error) {
	prov, err := d.provisionerFactory(ctx)
	if err != nil {
		return nil, err
	}
	status, err := prov.Create(ctx)
	if err != nil {
		return nil, err
	}
	status, err = prov.WaitForRunning(ctx, DefaultWaitTimeout)
	if err != nil {
		return nil, err
	}
	return driverResultFromStatus(status)
}

func (d *NodeDriver) GetState(ctx context.Context) (*DriverResult, error) {
	prov, err := d.provisionerFactory(ctx)
	if err != nil {
		return nil, err
	}
	status, err := prov.Status(ctx)
	if err != nil {
		return nil, err
	}
	return driverResultFromStatus(status)
}

func (d *NodeDriver) Remove(ctx context.Context) error {
	prov, err := d.provisionerFactory(ctx)
	if err != nil {
		return err
	}
	return prov.Destroy(ctx)
}

func (c EvrocMachineConfig) toMachineSpec() (machineSpec, error) {
	spec := machineSpec{
		NamePrefix:              c.NamePrefix,
		ProjectID:               c.ProjectID,
		Region:                  c.Region,
		Zone:                    c.Zone,
		ComputeProfileRef:       c.ComputeProfileRef,
		DiskImageRef:            c.DiskImageRef,
		DiskSizeGB:              c.DiskSizeGB,
		SSHAuthorizedKeys:       append([]string(nil), c.SSHAuthorizedKeys...),
		SSHSourceCIDR:           c.SSHSourceCIDR,
		KubernetesAPISourceCIDR: c.KubernetesAPISourceCIDR,
	}
	return spec, nil
}

func driverResultFromStatus(status *Status) (*DriverResult, error) {
	if status == nil {
		return &DriverResult{State: DriverStateMissing}, nil
	}
	result := &DriverResult{
		ProjectID: status.ProjectID,
		Region:    status.Region,
		Zone:      status.Zone,
		State:     DriverStateMissing,
	}
	if status.VirtualMachine == nil {
		return result, nil
	}
	result.NodeName = status.VirtualMachine.Metadata.ID
	result.VMState = status.VirtualMachine.Status.VirtualMachineStatus
	result.PrivateIP = status.VirtualMachine.Status.Networking.PrivateIPv4Address
	if status.PublicIP != nil {
		result.PublicIP = status.PublicIP.Status.PublicIPv4Address
		if result.PublicIP != "" {
			result.SSHCommand = fmt.Sprintf("ssh ubuntu@%s", result.PublicIP)
		}
	}
	switch status.VirtualMachine.Status.VirtualMachineStatus {
	case "Running":
		result.State = DriverStateRunning
	default:
		result.State = DriverStateProvisioning
	}
	return result, nil
}

func looksLikeSSHPublicKey(key string) bool {
	parts := strings.Fields(key)
	if len(parts) < 2 {
		return false
	}
	switch parts[0] {
	case "ssh-ed25519", "ssh-rsa", "ecdsa-sha2-nistp256", "ecdsa-sha2-nistp384", "ecdsa-sha2-nistp521":
		return true
	default:
		return false
	}
}
