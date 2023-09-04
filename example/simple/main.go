package main

import (
	"context"
	"log"

	"github.com/fango6/sshd"
	"golang.org/x/crypto/ssh"
)

func main() {
	mux := sshd.NewServeMux()
	mux.HandleFunc("session", func(ctx context.Context, conn *ssh.ServerConn, newChannel ssh.NewChannel) error {
		log.Printf("recv %s connection\n", conn.RemoteAddr())
		return conn.Close()
	})

	sshd.PublicKeyAuth(func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
		fingerprint := ssh.FingerprintSHA256(key)
		log.Printf("remote %s@%s, fingerprint %s", conn.User(), conn.RemoteAddr(), fingerprint)

		return &ssh.Permissions{
			Extensions: map[string]string{
				"fingerprint": fingerprint,
			},
		}, nil
	})

	if err := sshd.ListenAndServe(":56789", sshd.GetDefaultSshServerConfig, mux); err != nil {
		log.Println("serve error:", err)
	}
	log.Println("sshd exited")
}
