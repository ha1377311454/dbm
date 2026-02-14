package connection

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/argon2"
)

// argon2Params Argon2 参数配置
type argon2Params struct {
	time    uint32 // 迭代次数
	memory  uint32 // 内存使用（KB）
	threads uint8  // 并行度
	keyLen  uint32 // 派生密钥长度
	saltLen uint32 // 盐值长度
}

// 默认 Argon2 参数（OWASP 推荐 2023）
var defaultParams = &argon2Params{
	time:    3,      // 迭代次数
	memory:  64 * 1024, // 64MB
	threads: 4,      // 并行线程数
	keyLen:  32,     // AES-256 需要 32 字节
	saltLen: 16,     // 盐值长度
}

// Encryptor 密码加密器
type Encryptor struct {
	masterKey string   // 主密钥（用于派生加密密钥）
	params    *argon2Params
}

// NewEncryptor 创建加密器
func NewEncryptor(masterKey string) (*Encryptor, error) {
	if masterKey == "" {
		return nil, errors.New("master key cannot be empty")
	}

	return &Encryptor{
		masterKey: masterKey,
		params:    defaultParams,
	}, nil
}

// deriveKey 使用 Argon2id 从主密钥和盐值派生加密密钥
func (e *Encryptor) deriveKey(salt []byte) []byte {
	// 使用固定的应用盐值 + 随机盐值
	appSalt := []byte("dbm-encryption-v1") // 应用级固定盐值
	combinedSalt := append(appSalt, salt...)

	return argon2.IDKey(
		[]byte(e.masterKey),
		combinedSalt,
		e.params.time,
		e.params.memory,
		e.params.threads,
		e.params.keyLen,
	)
}

// Encrypt 加密密码
// 返回格式: base64(salt + nonce + ciphertext)
func (e *Encryptor) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	// 生成随机盐值
	salt := make([]byte, e.params.saltLen)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	// 使用盐值派生密钥
	key := e.deriveKey(salt)

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// 生成随机 nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// 加密数据
	ciphertext := gcm.Seal(nil, nonce, []byte(plaintext), nil)

	// 组合: salt + nonce + ciphertext
	result := append(salt, nonce...)
	result = append(result, ciphertext...)

	return base64.StdEncoding.EncodeToString(result), nil
}

// Decrypt 解密密码
func (e *Encryptor) Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	saltLen := int(e.params.saltLen)
	if len(data) < saltLen {
		return "", errors.New("ciphertext too short: missing salt")
	}

	// 提取盐值
	salt := data[:saltLen]
	data = data[saltLen:]

	// 使用盐值派生密钥
	key := e.deriveKey(salt)

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short: missing nonce")
	}

	// 提取 nonce 和密文
	nonce, cipherData := data[:nonceSize], data[nonceSize:]

	// 解密
	plaintext, err := gcm.Open(nil, nonce, cipherData, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}
