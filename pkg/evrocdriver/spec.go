package evrocdriver

import "fmt"

type machineSpec struct {
	NamePrefix              string
	ProjectID               string
	Region                  string
	Zone                    string
	ComputeProfileRef       string
	DiskImageRef            string
	DiskSizeGB              int32
	SSHAuthorizedKeys       []string
	SSHSourceCIDR           string
	KubernetesAPISourceCIDR string
}

type resourceNames struct {
	SecurityGroup  string
	PublicIP       string
	Disk           string
	VirtualMachine string
}

func (s machineSpec) names() resourceNames {
	return resourceNames{
		SecurityGroup:  fmt.Sprintf("%s-sg", s.NamePrefix),
		PublicIP:       fmt.Sprintf("%s-ip", s.NamePrefix),
		Disk:           fmt.Sprintf("%s-boot", s.NamePrefix),
		VirtualMachine: fmt.Sprintf("%s-node", s.NamePrefix),
	}
}
