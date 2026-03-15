package drivercli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/akmalabbasov/evroc-sdk/client"
	evrocconfig "github.com/akmalabbasov/evroc-sdk/config"
	evrocdriver "github.com/akmalabbasov/rancher-node-driver-evroc/pkg/evrocdriver"
	"gopkg.in/yaml.v3"
)

type Config struct {
	NamePrefix              string   `yaml:"namePrefix"`
	ConfigPath              string   `yaml:"configPath"`
	Profile                 string   `yaml:"profile"`
	ProjectID               string   `yaml:"projectID"`
	Region                  string   `yaml:"region"`
	Zone                    string   `yaml:"zone"`
	APIURL                  string   `yaml:"apiURL"`
	ComputeProfileRef       string   `yaml:"computeProfileRef"`
	DiskImageRef            string   `yaml:"diskImageRef"`
	DiskSizeGB              int32    `yaml:"diskSizeGB"`
	SSHAuthorizedKeys       []string `yaml:"sshAuthorizedKeys"`
	SSHSourceCIDR           string   `yaml:"sshSourceCIDR"`
	KubernetesAPISourceCIDR string   `yaml:"kubernetesAPISourceCIDR"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if cfg.NamePrefix == "" {
		cfg.NamePrefix = "rancher-poc"
	}
	if cfg.Profile == "" {
		cfg.Profile = "default"
	}
	if cfg.Region == "" {
		cfg.Region = "se-sto"
	}
	if cfg.Zone == "" {
		cfg.Zone = "a"
	}
	if cfg.DiskSizeGB == 0 {
		cfg.DiskSizeGB = 30
	}
	if cfg.SSHSourceCIDR == "" {
		cfg.SSHSourceCIDR = "0.0.0.0/0"
	}
	if cfg.KubernetesAPISourceCIDR == "" {
		cfg.KubernetesAPISourceCIDR = "0.0.0.0/0"
	}
	if cfg.ComputeProfileRef == "" {
		cfg.ComputeProfileRef = "/compute/global/computeProfiles/a1a.s"
	}
	if cfg.DiskImageRef == "" {
		cfg.DiskImageRef = "/compute/global/diskImages/evroc/ubuntu.24-04.1"
	}
	if cfg.ProjectID == "" {
		return nil, fmt.Errorf("projectID is required")
	}
	if len(cfg.SSHAuthorizedKeys) == 0 {
		return nil, fmt.Errorf("sshAuthorizedKeys must contain at least one key")
	}

	return &cfg, nil
}

func LoadMachineConfig(path string) (*evrocdriver.EvrocMachineConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read machine config: %w", err)
	}

	var cfg evrocdriver.EvrocMachineConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse machine config: %w", err)
	}
	return &cfg, nil
}

func (c *Config) EvrocConfigPath() (string, error) {
	if c.ConfigPath != "" {
		return c.ConfigPath, nil
	}
	if envPath := os.Getenv("EVROC_CONFIG_PATH"); envPath != "" {
		return envPath, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, ".evroc", "config.yaml"), nil
}

func (c *Config) NewClient(ctx context.Context) (*client.Client, error) {
	configPath, err := c.EvrocConfigPath()
	if err != nil {
		return nil, err
	}

	evrocCfg, err := evrocconfig.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("load evroc config: %w", err)
	}

	profileName := c.Profile
	if profileName == "" {
		profileName = evrocCfg.CurrentProfile
	}
	profile, err := evrocCfg.GetProfile(profileName)
	if err != nil {
		return nil, fmt.Errorf("resolve profile %q: %w", profileName, err)
	}

	apiURL := c.APIURL
	if apiURL == "" {
		apiURL = profile.APIURL
	}
	if apiURL == "" {
		apiURL = client.BaseURL
	}

	projectID := c.ProjectID
	if projectID == "" {
		projectID = profile.Project
	}
	region := c.Region
	if region == "" {
		region = profile.Region
	}

	var accessToken string
	switch {
	case profile.User.AccessToken != "":
		accessToken = profile.User.AccessToken
	case profile.User.RefreshToken != "":
		accessToken, err = client.ExchangeRefreshToken(ctx, profile.IssuerURL, profile.User.RefreshToken)
		if err != nil {
			return nil, fmt.Errorf("exchange refresh token: %w", err)
		}
	default:
		return nil, fmt.Errorf("profile %q has neither accessToken nor refreshToken", profileName)
	}

	return client.NewClient(accessToken, projectID, region, apiURL), nil
}

func (c *Config) MachineConfig() evrocdriver.EvrocMachineConfig {
	return evrocdriver.EvrocMachineConfig{
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
}
