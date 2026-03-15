package machinedriver

import (
	"context"
	"errors"
	"testing"

	evrocdriver "github.com/akmalabbasov/rancher-node-driver-evroc/pkg/evrocdriver"
)

type fakeBackend struct {
	result *evrocdriver.DriverResult
	err    error
	host   string
	user   string
	cmd    string
}

func (f *fakeBackend) Create(context.Context) (*evrocdriver.DriverResult, error) {
	return f.result, f.err
}
func (f *fakeBackend) Remove(context.Context) error { return f.err }
func (f *fakeBackend) GetState(context.Context) (*evrocdriver.DriverResult, error) {
	return f.result, f.err
}
func (f *fakeBackend) GetSSHHostname(context.Context) (string, error) { return f.host, f.err }
func (f *fakeBackend) GetSSHUsername() string                         { return f.user }
func (f *fakeBackend) GetSSHCommand(context.Context) (string, error)  { return f.cmd, f.err }

func TestMapState(t *testing.T) {
	if got := mapState(evrocdriver.DriverStateRunning); got != StateRunning {
		t.Fatalf("expected running, got %q", got)
	}
	if got := mapState(evrocdriver.DriverStateProvisioning); got != StateProvisioning {
		t.Fatalf("expected provisioning, got %q", got)
	}
	if got := mapState(evrocdriver.DriverStateMissing); got != StateMissing {
		t.Fatalf("expected missing, got %q", got)
	}
}

func TestRuntimeDriverGetResult(t *testing.T) {
	runtime := New(&fakeBackend{
		result: &evrocdriver.DriverResult{
			NodeName:   "node-1",
			State:      evrocdriver.DriverStateRunning,
			PublicIP:   "203.0.113.2",
			PrivateIP:  "10.0.0.5",
			SSHCommand: "ssh ubuntu@203.0.113.2",
			VMState:    "Running",
			ProjectID:  "project-1",
			Region:     "se-sto",
			Zone:       "a",
		},
	})

	result, err := runtime.GetResult(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.State != StateRunning {
		t.Fatalf("expected running state, got %q", result.State)
	}
	if result.SSHCommand != "ssh ubuntu@203.0.113.2" {
		t.Fatalf("unexpected ssh command %q", result.SSHCommand)
	}
	if result.PrivateIP != "10.0.0.5" {
		t.Fatalf("unexpected private IP %q", result.PrivateIP)
	}
}

func TestRuntimeDriverSSHAccessors(t *testing.T) {
	runtime := New(&fakeBackend{host: "203.0.113.3", user: "ubuntu", cmd: "ssh ubuntu@203.0.113.3"})

	host, err := runtime.GetSSHHostname(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if host != "203.0.113.3" {
		t.Fatalf("unexpected host %q", host)
	}
	if user := runtime.GetSSHUsername(); user != "ubuntu" {
		t.Fatalf("unexpected user %q", user)
	}
	cmd, err := runtime.GetSSHCommand(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cmd != "ssh ubuntu@203.0.113.3" {
		t.Fatalf("unexpected ssh command %q", cmd)
	}
}

func TestRuntimeDriverStateErrorPassesThrough(t *testing.T) {
	expected := errors.New("boom")
	runtime := New(&fakeBackend{err: expected})

	_, err := runtime.GetState(context.Background())
	if !errors.Is(err, expected) {
		t.Fatalf("expected %v, got %v", expected, err)
	}
}
