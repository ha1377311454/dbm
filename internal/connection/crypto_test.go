package connection

import (
	"strings"
	"testing"
)

func TestNewEncryptor(t *testing.T) {
	tests := []struct {
		name      string
		masterKey string
		wantErr   bool
	}{
		{
			name:      "valid key",
			masterKey: "test-master-key-123",
			wantErr:   false,
		},
		{
			name:      "empty key",
			masterKey: "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewEncryptor(tt.masterKey)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewEncryptor() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEncryptor_EncryptDecrypt(t *testing.T) {
	masterKey := "test-master-key-abc123"
	enc, err := NewEncryptor(masterKey)
	if err != nil {
		t.Fatalf("NewEncryptor() failed: %v", err)
	}

	tests := []struct {
		name      string
		plaintext string
	}{
		{
			name:      "normal password",
			plaintext: "my-secret-password-123",
		},
		{
			name:      "empty string",
			plaintext: "",
		},
		{
			name:      "special characters",
			plaintext: "!@#$%^&*()_+-=[]{}|;':\",./<>?",
		},
		{
			name:      "unicode characters",
			plaintext: "密码🔑测试🚀",
		},
		{
			name:      "long password",
			plaintext: strings.Repeat("a", 1000),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 加密
			ciphertext, err := enc.Encrypt(tt.plaintext)
			if err != nil {
				t.Errorf("Encrypt() error = %v", err)
				return
			}

			// 空字符串应该返回空
			if tt.plaintext == "" && ciphertext != "" {
				t.Errorf("Encrypt() empty plaintext should return empty, got %q", ciphertext)
				return
			}

			// 非空字符串应该返回非空的密文
			if tt.plaintext != "" && ciphertext == "" {
				t.Errorf("Encrypt() should return non-empty ciphertext")
				return
			}

			// 密文应该与明文不同
			if tt.plaintext != "" && ciphertext == tt.plaintext {
				t.Errorf("Encrypt() ciphertext should differ from plaintext")
				return
			}

			// 解密
			decrypted, err := enc.Decrypt(ciphertext)
			if err != nil {
				t.Errorf("Decrypt() error = %v", err)
				return
			}

			// 验证解密结果
			if decrypted != tt.plaintext {
				t.Errorf("Decrypt() = %q, want %q", decrypted, tt.plaintext)
			}
		})
	}
}

func TestEncryptor_DecryptInvalid(t *testing.T) {
	masterKey := "test-master-key-xyz"
	enc, err := NewEncryptor(masterKey)
	if err != nil {
		t.Fatalf("NewEncryptor() failed: %v", err)
	}

	tests := []struct {
		name      string
		ciphertext string
		wantErr   bool
	}{
		{
			name:      "invalid base64",
			ciphertext: "not-valid-base64!!!",
			wantErr:   true,
		},
		{
			name:      "truncated ciphertext",
			ciphertext: "YWJj", // valid base64 but too short
			wantErr:   true,
		},
		{
			name:      "wrong key",
			ciphertext: "wrong-ciphertext-data",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := enc.Decrypt(tt.ciphertext)
			if (err != nil) != tt.wantErr {
				t.Errorf("Decrypt() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEncryptor_DifferentKeys(t *testing.T) {
	plaintext := "my-password"

	enc1, _ := NewEncryptor("key-1")
	enc2, _ := NewEncryptor("key-2")

	ciphertext1, _ := enc1.Encrypt(plaintext)
	ciphertext2, _ := enc2.Encrypt(plaintext)

	// 相同明文用不同密钥加密，结果应该不同
	if ciphertext1 == ciphertext2 {
		t.Errorf("Different keys should produce different ciphertexts")
	}

	// 用错误密钥解密应该失败
	_, err := enc1.Decrypt(ciphertext2)
	if err == nil {
		t.Errorf("Decrypting with wrong key should fail")
	}
}

func TestEncryptor_DifferentSalts(t *testing.T) {
	masterKey := "test-key"
	enc, _ := NewEncryptor(masterKey)
	plaintext := "my-password"

	// 两次加密应该产生不同的密文（因为盐值是随机的）
	ciphertext1, _ := enc.Encrypt(plaintext)
	ciphertext2, _ := enc.Encrypt(plaintext)

	if ciphertext1 == ciphertext2 {
		t.Errorf("Each encryption should produce unique ciphertext (different salts)")
	}

	// 但解密都应该得到相同的明文
	dec1, _ := enc.Decrypt(ciphertext1)
	dec2, _ := enc.Decrypt(ciphertext2)

	if dec1 != plaintext || dec2 != plaintext {
		t.Errorf("Decryption should recover original plaintext")
	}
}

// 基准测试
func BenchmarkEncryptor_Encrypt(b *testing.B) {
	masterKey := "benchmark-key"
	enc, _ := NewEncryptor(masterKey)
	plaintext := "test-password-for-benchmarking"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = enc.Encrypt(plaintext)
	}
}

func BenchmarkEncryptor_Decrypt(b *testing.B) {
	masterKey := "benchmark-key"
	enc, _ := NewEncryptor(masterKey)
	plaintext := "test-password-for-benchmarking"
	ciphertext, _ := enc.Encrypt(plaintext)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = enc.Decrypt(ciphertext)
	}
}