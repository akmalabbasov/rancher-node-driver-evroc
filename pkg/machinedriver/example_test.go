package machinedriver

import (
	"strings"
	"testing"
)

func TestExampleYAMLContainsExpectedFields(t *testing.T) {
	out := ExampleYAML()
	for _, want := range []string{
		"namePrefix:",
		"projectID:",
		"region: se-sto",
		"sshAuthorizedKeys:",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected output to contain %q, got:\n%s", want, out)
		}
	}
}
