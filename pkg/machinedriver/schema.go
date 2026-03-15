package machinedriver

type FieldType string

const (
	FieldTypeString     FieldType = "string"
	FieldTypeInt        FieldType = "int"
	FieldTypeStringList FieldType = "stringList"
)

type FieldSpec struct {
	Name         string    `json:"name"`
	Type         FieldType `json:"type"`
	Label        string    `json:"label"`
	Description  string    `json:"description"`
	Required     bool      `json:"required"`
	DefaultValue any       `json:"defaultValue,omitempty"`
	Secret       bool      `json:"secret,omitempty"`
	MinValue     *int      `json:"minValue,omitempty"`
	Examples     []string  `json:"examples,omitempty"`
}

type MachineConfigSchema struct {
	DriverName string      `json:"driverName"`
	Fields     []FieldSpec `json:"fields"`
}

func Schema() MachineConfigSchema {
	return MachineConfigSchema{
		DriverName: "evroc",
		Fields: []FieldSpec{
			{Name: "namePrefix", Type: FieldTypeString, Label: "Name Prefix", Description: "Prefix used when naming Evroc resources for the machine.", Required: true, Examples: []string{"rancher-machine-poc"}},
			{Name: "projectID", Type: FieldTypeString, Label: "Project ID", Description: "Evroc project ID where the machine will be created.", Required: true, Examples: []string{"my-project-id"}},
			{Name: "region", Type: FieldTypeString, Label: "Region", Description: "Evroc region for the machine and its supporting resources.", Required: false, DefaultValue: "se-sto", Examples: []string{"se-sto"}},
			{Name: "zone", Type: FieldTypeString, Label: "Zone", Description: "Evroc availability zone for the machine boot disk and VM placement.", Required: false, DefaultValue: "a", Examples: []string{"a", "b", "c"}},
			{Name: "computeProfileRef", Type: FieldTypeString, Label: "Compute Profile", Description: "Evroc compute profile reference for the VM size.", Required: false, DefaultValue: "/compute/global/computeProfiles/a1a.s", Examples: []string{"/compute/global/computeProfiles/a1a.s"}},
			{Name: "diskImageRef", Type: FieldTypeString, Label: "Disk Image", Description: "Evroc disk image reference for the boot disk.", Required: false, DefaultValue: "/compute/global/diskImages/evroc/ubuntu.24-04.1", Examples: []string{"/compute/global/diskImages/evroc/ubuntu.24-04.1"}},
			{Name: "diskSizeGB", Type: FieldTypeInt, Label: "Disk Size (GB)", Description: "Boot disk size in GB.", Required: false, DefaultValue: 30, MinValue: intPtr(1)},
			{Name: "sshAuthorizedKeys", Type: FieldTypeStringList, Label: "SSH Authorized Keys", Description: "SSH public keys injected into the node for bootstrap and maintenance.", Required: true, Examples: []string{"ssh-ed25519 AAAA... user@example"}},
			{Name: "sshSourceCIDR", Type: FieldTypeString, Label: "SSH Source CIDR", Description: "CIDR allowed to access the node over SSH.", Required: false, DefaultValue: "0.0.0.0/0"},
			{Name: "kubernetesAPISourceCIDR", Type: FieldTypeString, Label: "Kubernetes API Source CIDR", Description: "CIDR allowed to access the Kubernetes API on the provisioned node.", Required: false, DefaultValue: "0.0.0.0/0"},
		},
	}
}

func intPtr(v int) *int { return &v }
