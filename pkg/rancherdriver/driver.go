package rancherdriver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/akmalabbasov/evroc-sdk/client"
	evrocdriver "github.com/akmalabbasov/rancher-node-driver-evroc/pkg/evrocdriver"
	"github.com/docker/machine/libmachine/drivers"
	"github.com/docker/machine/libmachine/mcnflag"
	"github.com/docker/machine/libmachine/ssh"
	"github.com/docker/machine/libmachine/state"
)

const (
	driverName               = "evroc"
	defaultRegion            = "se-sto"
	defaultZone              = "a"
	defaultComputeProfileRef = "/compute/global/computeProfiles/a1a.s"
	defaultDiskImageRef      = "/compute/global/diskImages/evroc/ubuntu.24-04.1"
	defaultDiskSizeGB        = 30
	defaultSSHUser           = "ubuntu"
	defaultSSHPort           = 22
	defaultEnginePort        = 2376
	defaultCIDR              = "0.0.0.0/0"
	defaultCreateTimeout     = 10 * time.Minute
)

type Driver struct {
	*drivers.BaseDriver

	APIURL                  string
	AccessToken             string
	ProjectID               string
	Region                  string
	Zone                    string
	ComputeProfileRef       string
	DiskImageRef            string
	DiskSizeGB              int
	SSHSourceCIDR           string
	KubernetesAPISourceCIDR string
	EnginePort              int
}

func NewDriver(hostName, storePath string) *Driver {
	return &Driver{
		BaseDriver: &drivers.BaseDriver{
			MachineName: hostName,
			StorePath:   storePath,
			SSHUser:     defaultSSHUser,
			SSHPort:     defaultSSHPort,
		},
		Region:                  defaultRegion,
		Zone:                    defaultZone,
		ComputeProfileRef:       defaultComputeProfileRef,
		DiskImageRef:            defaultDiskImageRef,
		DiskSizeGB:              defaultDiskSizeGB,
		SSHSourceCIDR:           defaultCIDR,
		KubernetesAPISourceCIDR: defaultCIDR,
		EnginePort:              defaultEnginePort,
	}
}

func (d *Driver) DriverName() string {
	return driverName
}

func (d *Driver) GetCreateFlags() []mcnflag.Flag {
	return []mcnflag.Flag{
		mcnflag.StringFlag{Name: "evroc-access-token", Usage: "Evroc API bearer token", EnvVar: "EVROC_ACCESS_TOKEN"},
		mcnflag.StringFlag{Name: "evroc-project-id", Usage: "Evroc project ID", EnvVar: "EVROC_PROJECT_ID"},
		mcnflag.StringFlag{Name: "evroc-api-url", Usage: "Evroc API base URL", Value: client.BaseURL, EnvVar: "EVROC_API_URL"},
		mcnflag.StringFlag{Name: "evroc-region", Usage: "Evroc region", Value: defaultRegion, EnvVar: "EVROC_REGION"},
		mcnflag.StringFlag{Name: "evroc-zone", Usage: "Evroc zone", Value: defaultZone, EnvVar: "EVROC_ZONE"},
		mcnflag.StringFlag{Name: "evroc-compute-profile-ref", Usage: "Evroc compute profile reference", Value: defaultComputeProfileRef, EnvVar: "EVROC_COMPUTE_PROFILE_REF"},
		mcnflag.StringFlag{Name: "evroc-disk-image-ref", Usage: "Evroc disk image reference", Value: defaultDiskImageRef, EnvVar: "EVROC_DISK_IMAGE_REF"},
		mcnflag.IntFlag{Name: "evroc-disk-size-gb", Usage: "Evroc boot disk size in GB", Value: defaultDiskSizeGB, EnvVar: "EVROC_DISK_SIZE_GB"},
		mcnflag.StringFlag{Name: "evroc-ssh-source-cidr", Usage: "CIDR allowed to SSH to the node", Value: defaultCIDR, EnvVar: "EVROC_SSH_SOURCE_CIDR"},
		mcnflag.StringFlag{Name: "evroc-kubernetes-api-source-cidr", Usage: "CIDR allowed to reach the Kubernetes API", Value: defaultCIDR, EnvVar: "EVROC_KUBERNETES_API_SOURCE_CIDR"},
		mcnflag.StringFlag{Name: "evroc-ssh-user", Usage: "SSH username for the created instance", Value: defaultSSHUser, EnvVar: "EVROC_SSH_USER"},
		mcnflag.IntFlag{Name: "evroc-ssh-port", Usage: "SSH port for the created instance", Value: defaultSSHPort, EnvVar: "EVROC_SSH_PORT"},
		mcnflag.IntFlag{Name: "evroc-engine-port", Usage: "Docker engine port used by machine", Value: defaultEnginePort, EnvVar: "EVROC_ENGINE_PORT"},
	}
}

func (d *Driver) SetConfigFromFlags(opts drivers.DriverOptions) error {
	d.AccessToken = opts.String("evroc-access-token")
	d.ProjectID = opts.String("evroc-project-id")
	d.APIURL = opts.String("evroc-api-url")
	d.Region = opts.String("evroc-region")
	d.Zone = opts.String("evroc-zone")
	d.ComputeProfileRef = opts.String("evroc-compute-profile-ref")
	d.DiskImageRef = opts.String("evroc-disk-image-ref")
	d.DiskSizeGB = opts.Int("evroc-disk-size-gb")
	d.SSHSourceCIDR = opts.String("evroc-ssh-source-cidr")
	d.KubernetesAPISourceCIDR = opts.String("evroc-kubernetes-api-source-cidr")
	d.SSHUser = opts.String("evroc-ssh-user")
	d.SSHPort = opts.Int("evroc-ssh-port")
	d.EnginePort = opts.Int("evroc-engine-port")

	if d.APIURL == "" {
		d.APIURL = client.BaseURL
	}
	if d.Region == "" {
		d.Region = defaultRegion
	}
	if d.Zone == "" {
		d.Zone = defaultZone
	}
	if d.ComputeProfileRef == "" {
		d.ComputeProfileRef = defaultComputeProfileRef
	}
	if d.DiskImageRef == "" {
		d.DiskImageRef = defaultDiskImageRef
	}
	if d.DiskSizeGB == 0 {
		d.DiskSizeGB = defaultDiskSizeGB
	}
	if d.SSHSourceCIDR == "" {
		d.SSHSourceCIDR = defaultCIDR
	}
	if d.KubernetesAPISourceCIDR == "" {
		d.KubernetesAPISourceCIDR = defaultCIDR
	}
	if d.SSHUser == "" {
		d.SSHUser = defaultSSHUser
	}
	if d.SSHPort == 0 {
		d.SSHPort = defaultSSHPort
	}
	if d.EnginePort == 0 {
		d.EnginePort = defaultEnginePort
	}

	return nil
}

func (d *Driver) PreCreateCheck() error {
	if d.AccessToken == "" {
		return errors.New("evroc-access-token is required")
	}
	if d.ProjectID == "" {
		return errors.New("evroc-project-id is required")
	}
	if d.MachineName == "" {
		return errors.New("machine name is required")
	}
	cfg, err := d.machineConfig(nil)
	if err != nil {
		return err
	}
	_, err = cfg.Normalize()
	return err
}

func (d *Driver) Create() error {
	if err := ssh.GenerateSSHKey(d.GetSSHKeyPath()); err != nil {
		return fmt.Errorf("generate ssh key: %w", err)
	}
	pubKey, err := os.ReadFile(d.GetSSHKeyPath() + ".pub")
	if err != nil {
		return fmt.Errorf("read ssh public key: %w", err)
	}
	prov, err := d.newProvisioner(pubKey)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultCreateTimeout)
	defer cancel()

	status, err := prov.Create(ctx)
	if err != nil {
		return err
	}
	status, err = prov.WaitForRunning(ctx, evrocdriver.DefaultWaitTimeout)
	if err != nil {
		return err
	}
	if err := prov.WaitForSSH(ctx, evrocdriver.DefaultSSHWaitTimeout); err != nil {
		return err
	}
	return d.applyStatus(status)
}

func (d *Driver) GetIP() (string, error) {
	if d.IPAddress != "" {
		return d.IPAddress, nil
	}
	st, err := d.getStatus()
	if err != nil {
		return "", err
	}
	if err := d.applyStatus(st); err != nil {
		return "", err
	}
	return d.BaseDriver.GetIP()
}

func (d *Driver) GetSSHHostname() (string, error) {
	return d.GetIP()
}

func (d *Driver) GetSSHUsername() string {
	if d.SSHUser == "" {
		return defaultSSHUser
	}
	return d.SSHUser
}

func (d *Driver) GetURL() (string, error) {
	if err := drivers.MustBeRunning(d); err != nil {
		return "", err
	}
	ip, err := d.GetIP()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("tcp://%s", net.JoinHostPort(ip, strconv.Itoa(d.EnginePort))), nil
}

func (d *Driver) GetState() (state.State, error) {
	status, err := d.getStatus()
	if err != nil {
		if client.IsNotFoundError(err) {
			return state.None, nil
		}
		return state.Error, err
	}
	if status == nil || status.VirtualMachine == nil {
		return state.None, nil
	}
	_ = d.applyStatus(status)

	switch status.VirtualMachine.Status.VirtualMachineStatus {
	case "Running":
		return state.Running, nil
	case "Stopped":
		return state.Stopped, nil
	case "Starting", "Provisioning":
		return state.Starting, nil
	case "Stopping":
		return state.Stopping, nil
	default:
		return state.None, nil
	}
}

func (d *Driver) Kill() error {
	return errors.New("evroc driver does not support kill")
}

func (d *Driver) Remove() error {
	prov, err := d.newProvisioner(nil)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultCreateTimeout)
	defer cancel()
	return prov.Destroy(ctx)
}

func (d *Driver) Restart() error {
	return errors.New("evroc driver does not support restart")
}

func (d *Driver) Start() error {
	current, err := d.GetState()
	if err != nil {
		return err
	}
	if current == state.Running {
		return nil
	}
	return errors.New("evroc driver does not support start")
}

func (d *Driver) Stop() error {
	return errors.New("evroc driver does not support stop")
}

func (d *Driver) machineConfig(authorizedKey []byte) (evrocdriver.EvrocMachineConfig, error) {
	keys := []string{}
	if len(authorizedKey) > 0 {
		keys = append(keys, strings.TrimSpace(string(authorizedKey)))
	} else if pubKey, err := os.ReadFile(d.GetSSHKeyPath() + ".pub"); err == nil {
		keys = append(keys, strings.TrimSpace(string(pubKey)))
	} else {
		keys = append(keys, "ssh-ed25519 AAAA... evroc-driver-placeholder")
	}
	cfg := evrocdriver.EvrocMachineConfig{
		NamePrefix:              sanitizeName(d.MachineName),
		ProjectID:               d.ProjectID,
		Region:                  d.Region,
		Zone:                    d.Zone,
		ComputeProfileRef:       d.ComputeProfileRef,
		DiskImageRef:            d.DiskImageRef,
		DiskSizeGB:              int32(d.DiskSizeGB),
		SSHAuthorizedKeys:       keys,
		SSHSourceCIDR:           d.SSHSourceCIDR,
		KubernetesAPISourceCIDR: d.KubernetesAPISourceCIDR,
	}
	return cfg.Normalize()
}

func (d *Driver) newProvisioner(authorizedKey []byte) (*evrocdriver.Provisioner, error) {
	cfg, err := d.machineConfig(authorizedKey)
	if err != nil {
		return nil, err
	}
	cli := client.NewClient(d.AccessToken, d.ProjectID, d.Region, d.APIURL)
	return evrocdriver.NewProvisioner(cli, cfg)
}

func (d *Driver) getStatus() (*evrocdriver.Status, error) {
	prov, err := d.newProvisioner(nil)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return prov.Status(ctx)
}

func (d *Driver) applyStatus(status *evrocdriver.Status) error {
	if status == nil || status.VirtualMachine == nil {
		return nil
	}
	if status.PublicIP != nil && status.PublicIP.Status.PublicIPv4Address != "" {
		d.IPAddress = status.PublicIP.Status.PublicIPv4Address
	}
	return nil
}

func sanitizeName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	var b strings.Builder
	lastDash := false
	for _, r := range name {
		isAlphaNum := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if isAlphaNum {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	s := strings.Trim(b.String(), "-")
	if s == "" {
		s = "evroc-machine"
	}
	if len(s) > 40 {
		s = strings.Trim(s[:40], "-")
	}
	return s
}
