package shared

import (
	"encoding/hex"
	"fmt"
	"github.com/mihb123/quanly-phongtro/pkg/apperr"
)

// DecodeEncryptionKey decodes the hex or raw 32-byte encryption key string used for Zalo tokens.
func DecodeEncryptionKey(hexKey string) ([]byte, error) {
	return DecodeAES256Key(hexKey, "ZALO_BOT_ENCRYPTION_KEY")
}

// DecodeAES256Key decodes a hex or raw 32-byte encryption key for AES-256-GCM.
func DecodeAES256Key(hexKey, keyName string) ([]byte, error) {
	keyBytes, err := hex.DecodeString(hexKey)
	if err != nil {
		if len(hexKey) == 32 {
			return []byte(hexKey), nil
		}
		return nil, apperr.Service("DecodeAES256Key", fmt.Errorf("invalid encryption key: %w", err))
	}
	if len(keyBytes) != 32 {
		return nil, apperr.Service("DecodeAES256Key", fmt.Errorf("%s must be exactly 32 bytes", keyName))
	}
	return keyBytes, nil
}
