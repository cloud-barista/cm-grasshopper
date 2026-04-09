package job

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
)

func MaskSecret(value string) string {
	if value == "" {
		return ""
	}
	if len(value) <= 8 {
		return strings.Repeat("*", len(value))
	}
	return value[:4] + strings.Repeat("*", len(value)-8) + value[len(value)-4:]
}

func KubeconfigFingerprint(value string) string {
	if value == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])[:12]
}

func BuildMetadataString(metadata map[string]interface{}) string {
	if len(metadata) == 0 {
		return ""
	}
	bytes, err := json.Marshal(metadata)
	if err != nil {
		return ""
	}
	return string(bytes)
}
