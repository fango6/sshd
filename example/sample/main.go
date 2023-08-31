package main

import (
	"context"
	"log"

	"github.com/fango6/sshd"
	"golang.org/x/crypto/ssh"
)

var noAuthSSConf = &ssh.ServerConfig{
	NoClientAuth: true,
}

func main() {
	mux := sshd.NewServeMux()
	mux.HandleFunc("session", func(ctx context.Context, conn *ssh.ServerConn, newChannel ssh.NewChannel) error {
		log.Printf("recv %s connection\n", conn.RemoteAddr())
		return conn.Close()
	})

	if err := sshd.ListenAndServe(":56789", getSshServerConfig, mux); err != nil {
		log.Println("serve error:", err)
	}
	log.Println("sshd exited")
}

func getSshServerConfig(ctx context.Context) *ssh.ServerConfig {
	return noAuthSSConf
}
