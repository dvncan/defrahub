package db

import (
	"fmt"

	"github.com/sourcenetwork/defradb/client"
)

// ExtractDocs pulls typed documents out of a DefraDB ExecRequest result.
func ExtractDocs(result *client.RequestResult, typeName string) ([]map[string]any, error) {
	if len(result.GQL.Errors) > 0 {
		return nil, fmt.Errorf("gql error: %v", result.GQL.Errors)
	}
	data, ok := result.GQL.Data.(map[string]any)
	if !ok {
		return nil, nil
	}
	docs := make([]map[string]any, 0)
	switch v := data[typeName].(type) {
	case []any:
		for _, r := range v {
			if m, ok := r.(map[string]any); ok {
				docs = append(docs, m)
			}
		}
	case []map[string]any:
		docs = append(docs, v...)
	default:
		return nil, nil
	}
	return docs, nil
}

// Str safely gets a string value from a document map.
func Str(m map[string]any, key string) string {
	v, _ := m[key].(string)
	return v
}

// Bool safely gets a bool value from a document map.
func Bool(m map[string]any, key string) bool {
	v, _ := m[key].(bool)
	return v
}

// Int64 safely gets an integer value from a document map.
func Int64(m map[string]any, key string) int64 {
	switch v := m[key].(type) {
	case float64:
		return int64(v)
	case int64:
		return v
	case int:
		return int64(v)
	}
	return 0
}
