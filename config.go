package sshd

import (
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
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
// 算法集与 golang.org/x/crypto/ssh 对应
func NewDefaultSshServerConfig() *ssh.ServerConfig {
	var hk ssh.Signer
	var err error

	for i := 0; i < 5; i++ {
		hk, err = GenerateEd25519HostKey()
		if err == nil && hk != nil {
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

func GenerateEd25519HostKey() (ssh.Signer, error) {
	_, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	return ssh.NewSignerFromKey(key)
}

func GenerateRsaHostKey(bits int) (ssh.Signer, error) {
	key, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, err
	}
	return ssh.NewSignerFromKey(key)
}

func GenerateEcdsaHostKey(curve elliptic.Curve) (ssh.Signer, error) {
	key, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		return nil, err
	}
	return ssh.NewSignerFromKey(key)
}

// ResolveHostKeys 读取 host key 文件, 并解析文件内容为 ssh.Signer.
func ResolveHostKeys(filepaths []string) ([]ssh.Signer, error) {
	return ResolveHostKeysWithDecode(filepaths, nil)
}

// ResolveHostKeysWithDecode 读取 host key 文件, 并解析文件内容.
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
