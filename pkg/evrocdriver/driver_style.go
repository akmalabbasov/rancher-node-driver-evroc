package evrocdriver

import "context"

func (d *NodeDriver) GetSSHHostname(ctx context.Context) (string, error) {
	result, err := d.GetState(ctx)
	if err != nil {
		return "", err
	}
	return result.PublicIP, nil
}

func (d *NodeDriver) GetSSHUsername() string {
	return "ubuntu"
}

func (d *NodeDriver) GetSSHCommand(ctx context.Context) (string, error) {
	result, err := d.GetState(ctx)
	if err != nil {
		return "", err
	}
	return result.SSHCommand, nil
}
