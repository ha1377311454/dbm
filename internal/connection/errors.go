package connection

import "errors"

var (
	// ErrConnectionNotFound 连接不存在
	ErrConnectionNotFound = errors.New("connection not found")

	// ErrInvalidConfig 无效的配置
	ErrInvalidConfig = errors.New("invalid connection config")

	// ErrEncryptionFailed 加密失败
	ErrEncryptionFailed = errors.New("encryption failed")

	// ErrDecryptionFailed 解密失败
	ErrDecryptionFailed = errors.New("decryption failed")
)
