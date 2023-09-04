package sshd

import (
	"net"

	"golang.org/x/crypto/ssh"
)

func Serve(ln net.Listener, fn GetSshServerConfig, handler Handler, options ...Option) error {
	return NewServer(fn, handler, options...).Serve(ln)
}

func ListenAndServe(addr string, fn GetSshServerConfig, handler Handler, options ...Option) error {
	srv := NewServer(fn, handler, options...)
	return srv.ListenAndServe(addr)
}

// PublicKeyAuth 通过公钥认证, 如果认证不通过, error 应返回非 nil.
func PublicKeyAuth(fn func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error)) {
	DefaultSshServerConfig.PublicKeyCallback = fn
}

// PasswordAuth 通过密码认证
func PasswordAuth(fn func(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error)) {
	DefaultSshServerConfig.PasswordCallback = fn
}

// KeyboardInteractiveAuth 通过交互方式认证
func KeyboardInteractiveAuth(fn func(conn ssh.ConnMetadata, client ssh.KeyboardInteractiveChallenge) (*ssh.Permissions, error)) {
	DefaultSshServerConfig.KeyboardInteractiveCallback = fn
}
