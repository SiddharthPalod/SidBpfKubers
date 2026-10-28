package security

import (
	"crypto/sha256"
	"encoding/hex"

	corev1 "k8s.io/api/core/v1"
)

// HashSecret computes a short SHA256 digest of secret data for rotation detection
func HashSecret(secret *corev1.Secret) string {
	if secret == nil {
		return ""
	}
	hasher := sha256.New()
	for k, v := range secret.Data {
		hasher.Write([]byte(k))
		hasher.Write(v)
	}
	return hex.EncodeToString(hasher.Sum(nil))[:16]
}
