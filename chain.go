package sshd

import (
	"context"
	"encoding/hex"
	"errors"
	"io"
	"math"
	"net"

	"github.com/anmitsu/go-shlex"
	"golang.org/x/crypto/ssh"
)

type Chain interface {
	context.Context
	io.Closer

	Next()
	Abort()
	IsAborted() bool

	User() string
	SessionID() string
	ClientVersion() string
	ServerVersion() string
	ClientIP() string
	ServerIP() string

	Stdin() io.Reader
	Stdout() io.Writer
	Stderr() io.Writer
	Exit(status uint32) error

	GetEnv(name string) string
}

type ChannelChain struct {
	context.Context

	ServerConfig *ssh.ServerConfig

	Conn        ssh.Conn
	Permissions *ssh.Permissions
	Channel     ssh.Channel

	Handler Handler

	index           int8
	commandHandlers []CommandHandler

	AcceptedEnvs map[string]string
	RawCommand   string
}

type (
	RequestHandler interface {
		Serve(cc *ChannelChain, req *ssh.Request) (ok bool, payload []byte)
	}
	RequestHandlerFunc func(cc *ChannelChain, req *ssh.Request) (ok bool, payload []byte)

	CommandHandler interface {
		Execute(cc *ChannelChain) error
	}
	CommandHandlerFunc func(cc *ChannelChain) error
)

// Serve implements RequestHandler.Serve
func (f RequestHandlerFunc) Serve(cc *ChannelChain, req *ssh.Request) (ok bool, payload []byte) {
	return f(cc, req)
}

func (f CommandHandlerFunc) Execute(cc *ChannelChain) error {
	return f(cc)
}

var _ Chain = &ChannelChain{}

func NewChannelChain(handler Handler, sc *ssh.ServerConfig) *ChannelChain {
	return &ChannelChain{
		ServerConfig: sc,
		Handler:      handler,
		AcceptedEnvs: make(map[string]string),
	}
}

func (cc *ChannelChain) entry(ctx context.Context, conn *ssh.ServerConn, newChannel ssh.NewChannel) error {
	cc.Context = ctx
	cc.Conn = conn.Conn
	cc.Permissions = conn.Permissions

	if cc.Handler == nil {
		newChannel.Reject(ssh.Prohibited, "prohibited any channel types")
		return errors.New("prohibited any channel types")
	}

	return cc.Handler.ServeChannel(cc, conn, newChannel)
}

func (cc *ChannelChain) HandleRequests(ch ssh.Channel, reqs <-chan *ssh.Request, handlers map[string]RequestHandler) {
	cc.Channel = ch

	for req := range reqs {
		if handlers == nil {
			req.Reply(false, nil)
			continue
		}

		handler, ok := handlers[req.Type]
		if !ok || handler == nil {
			handler, ok = handlers["default"]
		}

		if ok && handler != nil {
			ok, payload := handler.Serve(cc, req)
			req.Reply(ok, payload)
			continue
		}

		req.Reply(false, nil)
	}
}

func (cc *ChannelChain) HandleCommand(cmd string, acceptedEnvs map[string]string, handlers []CommandHandler) {
	cc.RawCommand = cmd
	cc.AcceptedEnvs = acceptedEnvs
	cc.index = -1
	cc.commandHandlers = handlers

	cc.Next()
}

func (cc *ChannelChain) Next() {
	cc.index++
	for cc.index < int8(len(cc.commandHandlers)) {
		cc.commandHandlers[cc.index].Execute(cc)
		cc.index++
	}
}

func (cc *ChannelChain) Abort() {
	cc.index = math.MaxInt8
}

func (cc *ChannelChain) IsAborted() bool {
	return cc.index >= int8(math.MaxInt8)
}

func (cc *ChannelChain) User() string {
	return cc.Conn.User()
}

func (cc *ChannelChain) SessionID() string {
	return hex.EncodeToString(cc.Conn.SessionID())
}

func (cc *ChannelChain) ClientVersion() string {
	return string(cc.Conn.ClientVersion())
}

func (cc *ChannelChain) ServerVersion() string {
	return string(cc.Conn.ServerVersion())
}

func (cc *ChannelChain) ClientIP() string {
	return cc.parseIP(cc.Conn.RemoteAddr())
}

func (cc *ChannelChain) ServerIP() string {
	return cc.parseIP(cc.Conn.LocalAddr())
}

func (cc *ChannelChain) parseIP(addr net.Addr) string {
	if addr == nil {
		return ""
	}

	ip := net.ParseIP(addr.String())
	if len(ip) == 0 {
		return ""
	}
	return ip.String()
}

func (cc *ChannelChain) PermExtensions(key string) string {
	if cc.Permissions == nil || cc.Permissions.Extensions == nil {
		return ""
	}
	val, ok := cc.Permissions.Extensions[key]
	if ok {
		return val
	}
	return ""
}

func (cc *ChannelChain) PermCriticalOptions(key string) string {
	if cc.Permissions == nil || cc.Permissions.CriticalOptions == nil {
		return ""
	}
	val, ok := cc.Permissions.CriticalOptions[key]
	if ok {
		return val
	}
	return ""
}

func (cc *ChannelChain) GetEnv(name string) string {
	if cc.AcceptedEnvs == nil {
		return ""
	}
	val, ok := cc.AcceptedEnvs[name]
	if ok {
		return val
	}
	return ""
}

func (cc *ChannelChain) SplitShellCmd(cmd string) []string {
	fields, err := shlex.Split(cmd, true)
	if err != nil {
		return nil
	}
	return fields
}

func (cc *ChannelChain) Stdin() io.Reader {
	return cc.Channel
}

func (cc *ChannelChain) Stdout() io.Writer {
	return cc.Channel
}

func (cc *ChannelChain) Stderr() io.Writer {
	return cc.Channel.Stderr()
}

func (cc *ChannelChain) Close() error {
	return cc.Channel.Close()
}

func (cc *ChannelChain) Exit(status uint32) error {
	var payload = struct {
		Status uint32
	}{
		Status: status,
	}

	_, err := cc.Channel.SendRequest("exit-status", false, ssh.Marshal(payload))
	if err != nil {
		return err
	}
	return cc.Channel.Close()
}
