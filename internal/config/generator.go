package config

import (
	"fmt"
	"reflect"
	"strings"

	"gopkg.in/yaml.v3"
)

// GenerateSkeleton generates a YAML skeleton from the Main configuration struct
// It uses reflection to iterate over all fields and create empty values
func GenerateSkeleton() ([]byte, error) {
	// Create a skeleton instance
	skeleton := generateStructSkeleton(reflect.TypeOf(Main{}))

	// Marshal to YAML with proper formatting
	data, err := yaml.Marshal(skeleton)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal skeleton to YAML: %w", err)
	}

	return data, nil
}

// generateStructSkeleton recursively generates a skeleton for a struct type
func generateStructSkeleton(t reflect.Type) interface{} {
	// Handle pointer types
	if t.Kind() == reflect.Ptr {
		return generateStructSkeleton(t.Elem())
	}

	// For non-struct types, return zero values
	if t.Kind() != reflect.Struct {
		return getZeroValue(t)
	}

	// Create a map to hold the struct fields
	result := make(map[string]interface{})

	// Iterate over all fields in the struct
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Get the YAML tag
		yamlTag := field.Tag.Get("yaml")
		if yamlTag == "" || yamlTag == "-" {
			continue
		}

		// Parse the YAML tag (format: "name,omitempty,inline")
		tagParts := strings.Split(yamlTag, ",")
		fieldName := tagParts[0]

		// Handle inline fields (embedded structs)
		if len(tagParts) > 1 && contains(tagParts[1:], "inline") {
			// For inline fields, merge the embedded struct fields
			inlineMap := generateStructSkeleton(field.Type)
			if m, ok := inlineMap.(map[string]interface{}); ok {
				for k, v := range m {
					result[k] = v
				}
			}
			continue
		}

		// Skip if field name is empty (shouldn't happen, but safe guard)
		if fieldName == "" {
			fieldName = toSnakeCase(field.Name)
		}

		// Generate the field value based on its type
		result[fieldName] = generateFieldValue(field.Type)
	}

	return result
}

// generateFieldValue generates an appropriate empty value for a field type
func generateFieldValue(t reflect.Type) interface{} {
	switch t.Kind() {
	case reflect.Ptr:
		// For pointer types, generate the underlying type
		return generateFieldValue(t.Elem())
	case reflect.Struct:
		// Recursively generate struct skeleton
		return generateStructSkeleton(t)
	case reflect.Slice, reflect.Array:
		// For slices/arrays, return an empty array with one example element
		elemType := t.Elem()
		if elemType.Kind() == reflect.Struct {
			// For struct slices, return array with one skeleton element
			return []interface{}{generateStructSkeleton(elemType)}
		}
		// For primitive slices, return empty array
		return []interface{}{}
	case reflect.Map:
		// Return empty map
		return map[string]interface{}{}
	default:
		// For primitive types, return zero value representation
		return getZeroValue(t)
	}
}

// getZeroValue returns a string representation of zero value for primitive types
func getZeroValue(t reflect.Type) interface{} {
	switch t.Kind() {
	case reflect.String:
		return ""
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return 0
	case reflect.Float32, reflect.Float64:
		return 0.0
	case reflect.Bool:
		return false
	case reflect.Interface:
		return ""
	default:
		return ""
	}
}

// contains checks if a string is in a slice
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// toSnakeCase converts a string to snake_case
// This is a fallback function for fields without YAML tags
func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			// Check if the previous character was uppercase
			// If so, we're in an acronym and don't need underscore
			if s[i-1] >= 'A' && s[i-1] <= 'Z' {
				// Check if next character is lowercase
				// If so, this is the start of a new word after an acronym
				if i+1 < len(s) && s[i+1] >= 'a' && s[i+1] <= 'z' {
					result.WriteRune('_')
				}
			} else {
				result.WriteRune('_')
			}
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}
