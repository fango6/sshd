package main

import (
	"context"
	"log"
	"net"

	"github.com/fango6/proxyproto"
	"github.com/fango6/sshd"
	"golang.org/x/crypto/ssh"
)

var defaultSshServerConf = &ssh.ServerConfig{
	ServerVersion: "SSH-2.0-GitLabShell",
}

func main() {
	mux := sshd.NewServeMux()
	mux.Handle("session", &SessionHandler{
		acceptEnvs: map[string]struct{}{
			"GIT_PROTOCOL": {},
		},
	})

	hostKey, err := sshd.GenerateEd25519HostKey()
	if err != nil {
		log.Fatalf("failed to generate host key, error:%v", err)
	}
	defaultSshServerConf.AddHostKey(hostKey)

	options := []sshd.Option{
		sshd.WithConnCallback(func(conn net.Conn) (newConn net.Conn) {
			return proxyproto.NewConn(conn)
		}),
		sshd.WithGetSshServerConfig(func(ctx context.Context) *ssh.ServerConfig {
			newSC := *defaultSshServerConf
			f := &Fingerprint{ctx: ctx}
			newSC.PublicKeyCallback = f.PublicKeyCallback
			return &newSC
		}),
	}

	err = sshd.ListenAndServe(":2222", mux, options...)
	log.Printf("bye now. error:%v\n", err)
}

type Fingerprint struct {
	ctx context.Context
}

func (f *Fingerprint) PublicKeyCallback(cm ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
	// do something
	return &ssh.Permissions{
		Extensions: map[string]string{
			"fingerprint":    ssh.FingerprintSHA256(key),
			"authorized_key": string(ssh.MarshalAuthorizedKey(key)),
		},
	}, nil
}

type SessionHandler struct {
	RequestHandlers map[string]sshd.RequestHandler
	CommandHandlers map[string]sshd.CommandHandler

	acceptEnvs   map[string]struct{}
	AcceptedEnvs map[string]string
}

func NewSessionHandler() *SessionHandler {
	sh := &SessionHandler{}
	sh.RequestHandlers = map[string]sshd.RequestHandler{
		"env":     sshd.RequestHandlerFunc(sh.EnvHandler),
		"exec":    sshd.RequestHandlerFunc(sh.ExecHandler),
		"shell":   sshd.RequestHandlerFunc(sh.ExecHandler),
		"default": sshd.RequestHandlerFunc(sh.DefaultHandler),
	}
	return sh
}

func (sh *SessionHandler) ServeChannel(cc *sshd.ChannelChain, sc *ssh.ServerConn, nc ssh.NewChannel) error {
	ch, reqs, err := nc.Accept()
	if err != nil {
		return err
	}

	cc.HandleRequests(ch, reqs, sh.RequestHandlers)
	return nil
}

func (sh *SessionHandler) EnvHandler(cc *sshd.ChannelChain, req *ssh.Request) (ok bool, payload []byte) {
	var env struct {
		Name  string
		Value string
	}
	if err := ssh.Unmarshal(req.Payload, &env); err != nil {
		log.Printf("failed to unmarshal env payload, error:%v\n", err)
		return false, nil
	}

	if _, ok := sh.acceptEnvs[env.Name]; ok {
		sh.AcceptedEnvs[env.Name] = env.Value
		return true, nil
	}
	return false, nil
}

func (sh *SessionHandler) ExecHandler(cc *sshd.ChannelChain, req *ssh.Request) (ok bool, payload []byte) {
	var exec struct {
		Command string
	}
	if err := ssh.Unmarshal(req.Payload, &exec); err != nil {
		log.Printf("failed to unmarshal command payload, error:%v\n", err)
		return false, nil
	}

	handlers := sh.GetCommandHandlers(cc.SplitShellCmd(exec.Command))
	cc.HandleCommand(exec.Command, sh.AcceptedEnvs, handlers)

	return true, nil
}

func (sh *SessionHandler) DefaultHandler(cc *sshd.ChannelChain, req *ssh.Request) (ok bool, payload []byte) {
	return false, nil
}

func (sh *SessionHandler) GetCommandHandlers(fields []string) []sshd.CommandHandler {
	// do something
	return []sshd.CommandHandler{
		sshd.CommandHandlerFunc(func(cc *sshd.ChannelChain) error {
			_, err := cc.Stderr().Write([]byte("not implements"))
			if err != nil {
				return err
			}
			return cc.Exit(1)
		}),
	}
}
