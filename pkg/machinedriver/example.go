package machinedriver

import (
	"fmt"
	"strings"
)

func ExampleYAML() string {
	schema := Schema()
	var b strings.Builder
	for _, field := range schema.Fields {
		switch field.Type {
		case FieldTypeString:
			value := stringValue(field)
			fmt.Fprintf(&b, "%s: %s\n", field.Name, value)
		case FieldTypeInt:
			fmt.Fprintf(&b, "%s: %v\n", field.Name, field.DefaultValue)
		case FieldTypeStringList:
			fmt.Fprintf(&b, "%s:\n", field.Name)
			examples := field.Examples
			if len(examples) == 0 {
				examples = []string{"example-value"}
			}
			for _, ex := range examples[:1] {
				fmt.Fprintf(&b, "  - %s\n", ex)
			}
		}
	}
	return b.String()
}

func stringValue(field FieldSpec) string {
	if field.DefaultValue != nil {
		if s, ok := field.DefaultValue.(string); ok {
			return s
		}
	}
	if len(field.Examples) > 0 {
		return field.Examples[0]
	}
	return ""
}
