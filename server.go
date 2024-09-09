package sshd

import (
	"context"
	"errors"
	"log"
	"net"
	"runtime"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

type (
	// ConnCallback 在接收 TCP 后, 对连接处理.
	ConnCallback func(conn net.Conn) (newConn net.Conn)

	// ConnContext 为 TCP 连接创建 context, 如在 context 中注入 uuid 等.
	ConnContext func(conn net.Conn) context.Context

	// GetSshServerConfig 获取 *ssh.ServerConfig 实例的函数, 将在 ssh 握手前调用.
	GetSshServerConfig func(ctx context.Context) *ssh.ServerConfig
)

// Server 支持为每一个 TCP 连接创建独立的 context, 确保在多次认证情况下 context 唯一.
// 支持 PROXY protocol, 并能够在处理 Channel 的请求时获取到 PROXY protocol 源数据.
type Server struct {
	ctx      context.Context
	cancel   context.CancelFunc
	once     sync.Once // once cancel
	connWG   sync.WaitGroup
	listener net.Listener

	// ConnCallback 在接收 TCP 后, 对连接处理, 如 PROXY protocol 等.
	ConnCallback ConnCallback

	// ConnContext 为 TCP 连接创建 context, 如在 context 中注入 uuid 等.
	// 将在建立 TCP 连接之后调用.
	ConnContext ConnContext

	// GetSshServerConfig 获取 *ssh.ServerConfig 实例的函数, 将在 ssh 握手前调用.
	// 入参的 context 来源于 ConnContext.
	GetSshServerConfig GetSshServerConfig

	// Handler 建立 ssh channel 时调用 Handler.ServeChannel.
	// 默认为 DefaultServeMux, 不会处理任何类型的 channel.
	Handler Handler

	// ReadTimeout 读超时时间, 在读取数据时重置读超时
	ReadTimeout time.Duration
	// WriteTimeout 写超时时间, 在写入数据时重置写超时
	WriteTimeout time.Duration
	// IdleTimeout 连接空闲时间, 默认为 30m, 在关闭 tcp 连接时设置读写超时
	IdleTimeout time.Duration

	// ErrLogger 输出捕获到的错误日志, 默认为 log.Default
	ErrLogger *log.Logger
}

func NewServer(handler Handler, options ...Option) *Server {
	ctx, cancel := context.WithCancel(context.Background())
	var srv = &Server{
		ctx:     ctx,
		cancel:  cancel,
		Handler: handler,
	}
	for _, opt := range options {
		opt(srv)
	}
	return srv
}

const maxStackInfoSize = 64 << 10

var (
	ErrServerClosed = errors.New("sshd: server closed")
)

// ListenAndServe 监听 TCP 连接, 如果 addr 为空字符串则监听地址为 ":2222"
func (srv *Server) ListenAndServe(addr string) error {
	if len(addr) == 0 {
		addr = ":2222"
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		srv.onceCancel()
		return err
	}

	return srv.Serve(ln)
}

func (srv *Server) Serve(ln net.Listener) error {
	srv.setDefaults()
	defer srv.onceCancel()

	srv.listener = ln

	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-srv.ctx.Done():
				return ErrServerClosed
			default:
				srv.logf("sshd: accept serving %s: %v", ln.Addr(), err)
			}
			continue
		}

		srv.connWG.Add(1)
		go srv.handshake(conn)
	}
}

func (srv *Server) handshake(conn net.Conn) {
	defer func() {
		if r := recover(); r != nil {
			buf := make([]byte, maxStackInfoSize)
			buf = buf[:runtime.Stack(buf, false)]
			srv.logf("sshd: panic serving %s: %v\n%s", conn.RemoteAddr(), r, buf)
			conn.Close()
		}
		srv.connWG.Done()
	}()

	if srv.ConnCallback != nil {
		conn = srv.ConnCallback(conn)
	}
	// spawn context for this connection
	var ctx = context.Background()
	if srv.ConnContext != nil {
		ctx = srv.ConnContext(conn)
	}
	newConn := &Conn{
		Conn:         conn,
		readTimeout:  srv.ReadTimeout,
		writeTimeout: srv.WriteTimeout,
		idleTimeout:  srv.IdleTimeout,
	}

	// ssh handshake
	ssConf := srv.GetSshServerConfig(ctx)
	sshConn, newChannels, reqs, err := ssh.NewServerConn(newConn, ssConf)
	if err != nil {
		srv.logf("sshd: handshake with %s error:%v", conn.RemoteAddr(), err)
		return
	}
	defer newConn.Close()

	// handle channels and requests
	go ssh.DiscardRequests(reqs)
	for newChannel := range newChannels {
		go srv.handleNewChannel(ctx, ssConf, sshConn, newChannel)
	}

	// close tcp connection
	if err := newConn.Close(); err != nil {
		_ = err
	}
}

func (srv *Server) handleNewChannel(ctx context.Context, conf *ssh.ServerConfig, conn *ssh.ServerConn, newChannel ssh.NewChannel) {
	defer func() {
		if r := recover(); r != nil {
			buf := make([]byte, maxStackInfoSize)
			buf = buf[:runtime.Stack(buf, false)]
			srv.logf("sshd: panic serving %s: %v\n%s", conn.RemoteAddr(), r, buf)
			conn.Close()
		}
	}()

	chain := NewChannelChain(srv.Handler, conf)
	err := chain.entry(ctx, conn, newChannel)
	if err != nil {
		srv.logf("sshd: serving %s the channel error: %v\n", conn.RemoteAddr(), err)
		return
	}
}

func (srv *Server) logf(format string, args ...interface{}) {
	if srv.ErrLogger != nil {
		srv.ErrLogger.Printf(format, args...)
	} else {
		log.Printf(format, args...)
	}
}

func (srv *Server) setDefaults() {
	if srv.ctx == nil {
		srv.ctx, srv.cancel = context.WithCancel(context.Background())
	} else if srv.ctx.Done() == nil || srv.cancel == nil {
		srv.ctx, srv.cancel = context.WithCancel(srv.ctx)
	}

	if srv.GetSshServerConfig == nil {
		srv.GetSshServerConfig = GetDefaultSshServerConfig
	}
	if srv.Handler == nil {
		srv.Handler = DefaultServeMux
	}
}

func (srv *Server) onceCancel() {
	srv.once.Do(func() {
		if srv.cancel != nil {
			srv.cancel()
		}
	})
}

// Shutdown 入参的 context, 如果非空则可用于优雅关闭
func (srv *Server) Shutdown(ctx context.Context) error {
	defer srv.onceCancel()

	if ctx == nil || ctx.Done() == nil {
		return srv.listener.Close()
	}
	<-ctx.Done()

	// graceful shutdown
	var done = make(chan error, 1)
	go func() {
		srv.connWG.Wait()

		if srv.listener == nil {
			done <- nil
			return
		}
		done <- srv.listener.Close()
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()

	case err := <-done:
		return err
	}
}
