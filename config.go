package sshd

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"
)

var DefaultSshServerConfig = NewDefaultSshServerConfig()

// GetDefaultSshServerConfig 获取默认的 ssh.ServerConfig, 同时适配 GetSshServerConfig
func GetDefaultSshServerConfig(_ context.Context) *ssh.ServerConfig {
	return DefaultSshServerConfig
}

// ReplaceDefaultSshServerConfig 替换默认的 ssh.ServerConfig
func ReplaceDefaultSshServerConfig(conf *ssh.ServerConfig) (original *ssh.ServerConfig) {
	original = DefaultSshServerConfig
	DefaultSshServerConfig = conf
	return original
}

// NewDefaultSshServerConfig 创建一个默认的 ssh.ServerConfig.
// Host key 类型为 rsa, bit size 为 3072.
//
// 以下算法集参考 golang.org/x/crypto v0.12.0
//
// Cipher 算法集为: aes128-gcm@openssh.com, aes256-gcm@openssh.com,
// chacha20-poly1305@openssh.com, aes128-ctr, aes192-ctr, aes256-ctr
//
// KEX (key exchange) 算法集为 curve25519-sha256, curve25519-sha256@libssh.org,
// ecdh-sha2-nistp256, ecdh-sha2-nistp384, ecdh-sha2-nistp521,
// diffie-hellman-group14-sha256, diffie-hellman-group14-sha1
//
// MAC (Message Authentication Code) 算法集为 hmac-sha2-256-etm@openssh.com,
// hmac-sha2-512-etm@openssh.com, hmac-sha2-256, hmac-sha2-512, hmac-sha1, hmac-sha1-96
func NewDefaultSshServerConfig() *ssh.ServerConfig {
	const retryTimes = 5
	var hk ssh.Signer
	var err error

	for i := 0; i < retryTimes; i++ {
		hk, err = generateSigner()
		if err == nil {
			break
		}
	}
	if err != nil {
		panic("sshd: failed to generate signer")
	}

	conf := &ssh.ServerConfig{}
	conf.AddHostKey(hk)
	conf.SetDefaults()
	return conf
}

func generateSigner() (ssh.Signer, error) {
	key, err := rsa.GenerateKey(rand.Reader, 3072)
	if err != nil {
		return nil, err
	}
	return ssh.NewSignerFromKey(key)
}

// ResolveHostKeys 读取 host key 文件, 并解析文件内容为 ssh.Signer.
func ResolveHostKeys(filepaths []string) ([]ssh.Signer, error) {
	return ResolveHostKeysWithDecode(filepaths, nil)
}

// ParseHostKeysWithDecodeFunc 读取 host key 文件, 并解析文件内容.
// 实现 decodeFunc 函数可对文件内容解码, 再解析为 ssh.Signer.
func ResolveHostKeysWithDecode(filepaths []string, decodeFunc func(pem []byte) ([]byte, error)) ([]ssh.Signer, error) {
	var hostKeys []ssh.Signer
	for _, filename := range filepaths {
		filename = filepath.Clean(filename)
		pem, err := os.ReadFile(filename)
		if err != nil {
			return nil, err
		}

		if decodeFunc != nil {
			pem, err = decodeFunc(pem)
			if err != nil {
				return nil, err
			}
		}

		signer, err := ssh.ParsePrivateKey(pem)
		if err != nil {
			return nil, err
		}
		hostKeys = append(hostKeys, signer)
	}
	return hostKeys, nil
}
