package machinedriver

import (
	"context"

	evrocdriver "github.com/akmalabbasov/rancher-node-driver-evroc/pkg/evrocdriver"
)

type Driver interface {
	Create(context.Context) error
	Remove(context.Context) error
	GetState(context.Context) (State, error)
	GetSSHHostname(context.Context) (string, error)
	GetSSHUsername() string
	GetSSHCommand(context.Context) (string, error)
	GetResult(context.Context) (*Result, error)
}

type backend interface {
	Create(context.Context) (*evrocdriver.DriverResult, error)
	Remove(context.Context) error
	GetState(context.Context) (*evrocdriver.DriverResult, error)
	GetSSHHostname(context.Context) (string, error)
	GetSSHUsername() string
	GetSSHCommand(context.Context) (string, error)
}

type State string

const (
	StateMissing      State = "missing"
	StateProvisioning State = "provisioning"
	StateRunning      State = "running"
)

type Result struct {
	NodeName   string `json:"nodeName"`
	State      State  `json:"state"`
	PublicIP   string `json:"publicIP,omitempty"`
	PrivateIP  string `json:"privateIP,omitempty"`
	SSHCommand string `json:"sshCommand,omitempty"`
	VMState    string `json:"vmState,omitempty"`
	ProjectID  string `json:"projectID,omitempty"`
	Region     string `json:"region,omitempty"`
	Zone       string `json:"zone,omitempty"`
}

type RuntimeDriver struct {
	driver backend
}

func New(driver backend) *RuntimeDriver {
	return &RuntimeDriver{driver: driver}
}

func (d *RuntimeDriver) Create(ctx context.Context) error {
	_, err := d.driver.Create(ctx)
	return err
}

func (d *RuntimeDriver) Remove(ctx context.Context) error {
	return d.driver.Remove(ctx)
}

func (d *RuntimeDriver) GetState(ctx context.Context) (State, error) {
	result, err := d.driver.GetState(ctx)
	if err != nil {
		return StateMissing, err
	}
	return mapState(result.State), nil
}

func (d *RuntimeDriver) GetSSHHostname(ctx context.Context) (string, error) {
	return d.driver.GetSSHHostname(ctx)
}

func (d *RuntimeDriver) GetSSHUsername() string {
	return d.driver.GetSSHUsername()
}

func (d *RuntimeDriver) GetSSHCommand(ctx context.Context) (string, error) {
	return d.driver.GetSSHCommand(ctx)
}

func (d *RuntimeDriver) GetResult(ctx context.Context) (*Result, error) {
	result, err := d.driver.GetState(ctx)
	if err != nil {
		return nil, err
	}
	return &Result{
		NodeName:   result.NodeName,
		State:      mapState(result.State),
		PublicIP:   result.PublicIP,
		PrivateIP:  result.PrivateIP,
		SSHCommand: result.SSHCommand,
		VMState:    result.VMState,
		ProjectID:  result.ProjectID,
		Region:     result.Region,
		Zone:       result.Zone,
	}, nil
}

func mapState(state evrocdriver.DriverState) State {
	switch state {
	case evrocdriver.DriverStateRunning:
		return StateRunning
	case evrocdriver.DriverStateProvisioning:
		return StateProvisioning
	default:
		return StateMissing
	}
}
