package machinedriver

import "testing"

func TestSchemaContainsExpectedFields(t *testing.T) {
	schema := Schema()
	if schema.DriverName != "evroc" {
		t.Fatalf("expected driver name evroc, got %q", schema.DriverName)
	}
	if len(schema.Fields) < 5 {
		t.Fatalf("expected several fields, got %d", len(schema.Fields))
	}

	required := map[string]bool{}
	for _, field := range schema.Fields {
		required[field.Name] = field.Required
	}
	if !required["namePrefix"] {
		t.Fatal("expected namePrefix to be required")
	}
	if !required["projectID"] {
		t.Fatal("expected projectID to be required")
	}
	if !required["sshAuthorizedKeys"] {
		t.Fatal("expected sshAuthorizedKeys to be required")
	}
}

func TestSchemaValidationHints(t *testing.T) {
	schema := Schema()
	fields := map[string]FieldSpec{}
	for _, field := range schema.Fields {
		fields[field.Name] = field
	}

	if fields["diskSizeGB"].MinValue == nil || *fields["diskSizeGB"].MinValue != 1 {
		t.Fatal("expected diskSizeGB to declare minValue 1")
	}
	if len(fields["zone"].Examples) == 0 {
		t.Fatal("expected zone field to include examples")
	}
	if len(fields["sshAuthorizedKeys"].Examples) == 0 {
		t.Fatal("expected sshAuthorizedKeys field to include examples")
	}
}
