package evrocdriver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/akmalabbasov/evroc-sdk/client"
)

const DefaultWaitTimeout = 90 * time.Second
const DefaultSSHWaitTimeout = 2 * time.Minute

type Provisioner struct {
	client *client.Client
	spec   machineSpec
	names  resourceNames
}

type Status struct {
	ProjectID      string                 `json:"projectID"`
	Region         string                 `json:"region"`
	Zone           string                 `json:"zone"`
	SecurityGroup  *client.SecurityGroup  `json:"securityGroup,omitempty"`
	PublicIP       *client.PublicIP       `json:"publicIP,omitempty"`
	Disk           *client.Disk           `json:"disk,omitempty"`
	VirtualMachine *client.VirtualMachine `json:"virtualMachine,omitempty"`
}

type CreateSummary struct {
	NodeName   string `json:"nodeName"`
	PublicIP   string `json:"publicIP,omitempty"`
	SSHCommand string `json:"sshCommand,omitempty"`
	VMState    string `json:"vmState,omitempty"`
	ProjectID  string `json:"projectID"`
	Region     string `json:"region"`
	Zone       string `json:"zone"`
}

func NewProvisioner(cli *client.Client, cfg EvrocMachineConfig) (*Provisioner, error) {
	spec, err := cfg.toMachineSpec()
	if err != nil {
		return nil, err
	}
	return NewProvisionerFromSpec(cli, spec), nil
}

func NewProvisionerFromSpec(cli *client.Client, spec machineSpec) *Provisioner {
	return &Provisioner{client: cli, spec: spec, names: spec.names()}
}

func (p *Provisioner) Create(ctx context.Context) (*Status, error) {
	if _, err := p.ensureSecurityGroup(ctx); err != nil {
		return nil, err
	}
	publicIP, err := p.ensurePublicIP(ctx)
	if err != nil {
		return nil, err
	}
	disk, err := p.ensureDisk(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := p.ensureVirtualMachine(ctx, publicIP, disk); err != nil {
		return nil, err
	}
	return p.Status(ctx)
}

func (p *Provisioner) Status(ctx context.Context) (*Status, error) {
	status := &Status{ProjectID: p.spec.ProjectID, Region: p.spec.Region, Zone: p.spec.Zone}

	if sg, err := p.client.GetSecurityGroup(ctx, p.spec.ProjectID, p.spec.Region, p.names.SecurityGroup); err == nil {
		status.SecurityGroup = sg
	} else if !client.IsNotFoundError(err) {
		return nil, err
	}
	if publicIP, err := p.client.GetPublicIP(ctx, p.spec.ProjectID, p.spec.Region, p.names.PublicIP); err == nil {
		status.PublicIP = publicIP
	} else if !client.IsNotFoundError(err) {
		return nil, err
	}
	if disk, err := p.client.GetDisk(ctx, p.spec.ProjectID, p.spec.Region, p.names.Disk); err == nil {
		status.Disk = disk
	} else if !client.IsNotFoundError(err) {
		return nil, err
	}
	if vm, err := p.client.GetVirtualMachine(ctx, p.spec.ProjectID, p.spec.Region, p.names.VirtualMachine); err == nil {
		status.VirtualMachine = vm
	} else if !client.IsNotFoundError(err) {
		return nil, err
	}

	return status, nil
}

func (p *Provisioner) Destroy(ctx context.Context) error {
	if err := deleteIfExists(ctx, func(ctx context.Context) error {
		return p.client.DeleteVirtualMachine(ctx, p.spec.ProjectID, p.spec.Region, p.names.VirtualMachine)
	}); err != nil {
		return fmt.Errorf("delete vm: %w", err)
	}
	if err := deleteIfExists(ctx, func(ctx context.Context) error {
		return p.client.DeleteDisk(ctx, p.spec.ProjectID, p.spec.Region, p.names.Disk)
	}); err != nil {
		return fmt.Errorf("delete disk: %w", err)
	}
	if err := deleteIfExists(ctx, func(ctx context.Context) error {
		return p.client.DeletePublicIP(ctx, p.spec.ProjectID, p.spec.Region, p.names.PublicIP)
	}); err != nil {
		return fmt.Errorf("delete public IP: %w", err)
	}
	if err := deleteIfExists(ctx, func(ctx context.Context) error {
		return p.client.DeleteSecurityGroup(ctx, p.spec.ProjectID, p.spec.Region, p.names.SecurityGroup)
	}); err != nil {
		return fmt.Errorf("delete security group: %w", err)
	}
	return nil
}

func (p *Provisioner) WaitForStatus(ctx context.Context, timeout time.Duration) (*Status, error) {
	return p.waitForCondition(ctx, timeout, readyForSummary)
}

func (p *Provisioner) WaitForRunning(ctx context.Context, timeout time.Duration) (*Status, error) {
	return p.waitForCondition(ctx, timeout, readyForRunning)
}

func (p *Provisioner) waitForCondition(ctx context.Context, timeout time.Duration, fn func(*Status) bool) (*Status, error) {
	deadline := time.Now().Add(timeout)
	for {
		status, err := p.Status(ctx)
		if err != nil {
			return nil, err
		}
		if fn(status) {
			return status, nil
		}
		if time.Now().After(deadline) {
			return status, nil
		}
		select {
		case <-ctx.Done():
			return status, ctx.Err()
		case <-time.After(3 * time.Second):
		}
	}
}

func (p *Provisioner) WaitForSSH(ctx context.Context, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		status, err := p.Status(ctx)
		if err != nil {
			return err
		}
		ip := ""
		if status.PublicIP != nil {
			ip = status.PublicIP.Status.PublicIPv4Address
		}
		if ip != "" {
			conn, err := net.DialTimeout("tcp", net.JoinHostPort(ip, "22"), 5*time.Second)
			if err == nil {
				_ = conn.Close()
				return nil
			}
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for SSH on %s", ip)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
		}
	}
}

func (p *Provisioner) Summary(status *Status) (*CreateSummary, error) {
	if status == nil || status.VirtualMachine == nil {
		return nil, fmt.Errorf("missing virtual machine in status")
	}

	ip := ""
	if status.PublicIP != nil {
		ip = status.PublicIP.Status.PublicIPv4Address
	}
	summary := &CreateSummary{
		NodeName:  status.VirtualMachine.Metadata.ID,
		PublicIP:  ip,
		VMState:   status.VirtualMachine.Status.VirtualMachineStatus,
		ProjectID: status.ProjectID,
		Region:    status.Region,
		Zone:      status.Zone,
	}
	if ip != "" {
		summary.SSHCommand = fmt.Sprintf("ssh ubuntu@%s", ip)
	}
	return summary, nil
}

func PrintJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func readyForSummary(status *Status) bool {
	if status == nil || status.PublicIP == nil || status.VirtualMachine == nil {
		return false
	}
	return status.PublicIP.Status.PublicIPv4Address != ""
}

func readyForRunning(status *Status) bool {
	if !readyForSummary(status) {
		return false
	}
	return status.VirtualMachine.Status.VirtualMachineStatus == "Running"
}

func deleteIfExists(ctx context.Context, fn func(context.Context) error) error {
	err := fn(ctx)
	if err == nil || client.IsNotFoundError(err) {
		return nil
	}
	var apiErr *client.APIError
	if errors.As(err, &apiErr) && apiErr.StatusCode == 409 {
		return nil
	}
	return err
}

func boolPtr(v bool) *bool { return &v }

func (p *Provisioner) ensureSecurityGroup(ctx context.Context) (*client.SecurityGroup, error) {
	if sg, err := p.client.GetSecurityGroup(ctx, p.spec.ProjectID, p.spec.Region, p.names.SecurityGroup); err == nil {
		return sg, nil
	} else if !client.IsNotFoundError(err) {
		return nil, err
	}

	sg := &client.SecurityGroup{APIVersion: "networking/v1beta1", Kind: "SecurityGroup"}
	sg.Metadata = client.Metadata{
		ID:      p.names.SecurityGroup,
		Project: p.spec.ProjectID,
		Region:  p.spec.Region,
		UserLabels: map[string]string{
			"managed-by": "rancher-poc",
		},
	}
	sg.Spec.Rules = []client.SecurityGroupRule{
		{Name: "allow-ssh", Direction: "Ingress", Protocol: "TCP", Port: 22, Remote: client.SecurityGroupRuleRemote{Address: &client.SecurityGroupRuleAddress{IPAddressOrCIDR: p.spec.SSHSourceCIDR}}},
		{Name: "allow-k8s-api", Direction: "Ingress", Protocol: "TCP", Port: 6443, Remote: client.SecurityGroupRuleRemote{Address: &client.SecurityGroupRuleAddress{IPAddressOrCIDR: p.spec.KubernetesAPISourceCIDR}}},
		{Name: "allow-self", Direction: "Ingress", Protocol: "All", Remote: client.SecurityGroupRuleRemote{SecurityGroupRef: fmt.Sprintf("/networking/projects/%s/regions/%s/securityGroups/%s", p.spec.ProjectID, p.spec.Region, p.names.SecurityGroup)}},
		{Name: "allow-egress", Direction: "Egress", Protocol: "All", Remote: client.SecurityGroupRuleRemote{Address: &client.SecurityGroupRuleAddress{IPAddressOrCIDR: "0.0.0.0/0"}}},
	}
	return p.client.CreateSecurityGroup(ctx, sg)
}

func (p *Provisioner) ensurePublicIP(ctx context.Context) (*client.PublicIP, error) {
	if publicIP, err := p.client.GetPublicIP(ctx, p.spec.ProjectID, p.spec.Region, p.names.PublicIP); err == nil {
		return publicIP, nil
	} else if !client.IsNotFoundError(err) {
		return nil, err
	}

	publicIP := &client.PublicIP{APIVersion: "networking/v1beta1", Kind: "PublicIP"}
	publicIP.Metadata = client.Metadata{ID: p.names.PublicIP, Project: p.spec.ProjectID, Region: p.spec.Region, UserLabels: map[string]string{"managed-by": "rancher-poc"}}
	return p.client.CreatePublicIP(ctx, publicIP)
}

func (p *Provisioner) ensureDisk(ctx context.Context) (*client.Disk, error) {
	if disk, err := p.client.GetDisk(ctx, p.spec.ProjectID, p.spec.Region, p.names.Disk); err == nil {
		return disk, nil
	} else if !client.IsNotFoundError(err) {
		return nil, err
	}

	disk := &client.Disk{APIVersion: "compute/v1beta1", Kind: "Disk"}
	disk.Metadata = client.Metadata{ID: p.names.Disk, Project: p.spec.ProjectID, Region: p.spec.Region, UserLabels: map[string]string{"managed-by": "rancher-poc"}}
	disk.Spec.DiskImageRef = p.spec.DiskImageRef
	disk.Spec.DiskSize = &client.DiskSize{Amount: p.spec.DiskSizeGB, Unit: "GB"}
	disk.Spec.Placement.Zone = p.spec.Zone
	return p.client.CreateDisk(ctx, disk)
}

func (p *Provisioner) ensureVirtualMachine(ctx context.Context, publicIP *client.PublicIP, disk *client.Disk) (*client.VirtualMachine, error) {
	if vm, err := p.client.GetVirtualMachine(ctx, p.spec.ProjectID, p.spec.Region, p.names.VirtualMachine); err == nil {
		return vm, nil
	} else if !client.IsNotFoundError(err) {
		return nil, err
	}

	vm := &client.VirtualMachine{APIVersion: "compute/v1beta1", Kind: "VirtualMachine"}
	vm.Metadata = client.Metadata{ID: p.names.VirtualMachine, Project: p.spec.ProjectID, Region: p.spec.Region, UserLabels: map[string]string{"managed-by": "rancher-poc", "role": "all-in-one"}}
	vm.Spec.ComputeProfileRef = p.spec.ComputeProfileRef
	vm.Spec.Disks = []client.VMDisk{{DiskRef: fmt.Sprintf("/compute/projects/%s/regions/%s/disks/%s", p.spec.ProjectID, p.spec.Region, disk.Metadata.ID), BootFrom: true}}
	vm.Spec.Placement.Zone = p.spec.Zone
	vm.Spec.Running = boolPtr(true)
	vm.Spec.Networking = &client.VMNetworking{
		PublicIPv4Address:     &client.VMPublicIPSettings{Static: &client.VMStaticPublicIP{PublicIPRef: fmt.Sprintf("/networking/projects/%s/regions/%s/publicIPs/%s", p.spec.ProjectID, p.spec.Region, publicIP.Metadata.ID)}},
		SecurityGroupSettings: &client.VMSecurityGroupSettings{SecurityGroupMemberRefs: []string{fmt.Sprintf("/networking/projects/%s/regions/%s/securityGroups/%s", p.spec.ProjectID, p.spec.Region, p.names.SecurityGroup)}},
	}
	vm.Spec.OSSettings = &client.OSSettings{CloudInitUserData: cloudInit(p.names.VirtualMachine)}
	if len(p.spec.SSHAuthorizedKeys) > 0 {
		vm.Spec.OSSettings.SSH = &client.SSH{AuthorizedKeys: make([]client.SSHKey, 0, len(p.spec.SSHAuthorizedKeys))}
		for _, key := range p.spec.SSHAuthorizedKeys {
			vm.Spec.OSSettings.SSH.AuthorizedKeys = append(vm.Spec.OSSettings.SSH.AuthorizedKeys, client.SSHKey{Value: key})
		}
	}
	return p.client.CreateVirtualMachine(ctx, vm)
}
