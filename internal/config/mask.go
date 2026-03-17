package config

import (
	"encoding/json"
	"strings"
)

var sensitiveKeySubstrings = []string{
	"apikey", "api_key",
	"secret",
	"token",
	"password", "passwd",
	"credential",
	"privatekey", "private_key",
	"authorization",
	"signingkey", "signing_key",
	"encryptionkey", "encryption_key",
}

func IsSensitiveKey(key string) bool {
	lower := strings.ToLower(key)
	for _, sub := range sensitiveKeySubstrings {
		if strings.Contains(lower, sub) {
			return true
		}
	}
	return false
}

func MaskSecret(val string) string {
	return MaskAPIKey(val)
}

// IsMaskedValue returns true if the string looks like it was produced by MaskSecret.
func IsMaskedValue(val string) bool {
	if val == "****" {
		return true
	}
	if len(val) == 10 && val[4:7] == "..." {
		return true
	}
	return false
}

// MaskSensitiveJSON recursively masks any string value whose key
// matches a sensitive pattern (apiKey, secret, token, password, etc.).
func MaskSensitiveJSON(data interface{}) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{}, len(v))
		for key, val := range v {
			if IsSensitiveKey(key) {
				if s, ok := val.(string); ok && s != "" {
					result[key] = MaskSecret(s)
					continue
				}
			}
			result[key] = MaskSensitiveJSON(val)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = MaskSensitiveJSON(item)
		}
		return result
	default:
		return data
	}
}

// RestoreSensitiveJSON walks new (possibly masked) JSON against the original
// and restores any sensitive values that still carry the mask placeholder.
func RestoreSensitiveJSON(newData, oldData interface{}) interface{} {
	switch newVal := newData.(type) {
	case map[string]interface{}:
		oldMap, ok := oldData.(map[string]interface{})
		if !ok {
			return newData
		}
		result := make(map[string]interface{}, len(newVal))
		for key, val := range newVal {
			if IsSensitiveKey(key) {
				if newStr, ok := val.(string); ok {
					if oldStr, ok := oldMap[key].(string); ok && oldStr != "" {
						if IsMaskedValue(newStr) {
							result[key] = oldStr
							continue
						}
					}
				}
			}
			if oldVal, exists := oldMap[key]; exists {
				result[key] = RestoreSensitiveJSON(val, oldVal)
			} else {
				result[key] = val
			}
		}
		return result
	case []interface{}:
		oldArr, ok := oldData.([]interface{})
		if !ok {
			return newData
		}
		result := make([]interface{}, len(newVal))
		for i, item := range newVal {
			if i < len(oldArr) {
				result[i] = RestoreSensitiveJSON(item, oldArr[i])
			} else {
				result[i] = item
			}
		}
		return result
	default:
		return newData
	}
}

// MaskRawJSON parses raw JSON, masks all sensitive fields, returns pretty-printed bytes.
func MaskRawJSON(data []byte) ([]byte, error) {
	var generic interface{}
	if err := json.Unmarshal(data, &generic); err != nil {
		return nil, err
	}
	masked := MaskSensitiveJSON(generic)
	return json.MarshalIndent(masked, "", "  ")
}

// RestoreRawJSON restores masked sensitive values in newContent by comparing
// against originalData, and returns the restored pretty-printed JSON.
func RestoreRawJSON(newContent, originalData []byte) ([]byte, error) {
	var newGeneric, oldGeneric interface{}
	if err := json.Unmarshal(newContent, &newGeneric); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(originalData, &oldGeneric); err != nil {
		return newContent, nil
	}
	restored := RestoreSensitiveJSON(newGeneric, oldGeneric)
	result, err := json.MarshalIndent(restored, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(result, '\n'), nil
}
