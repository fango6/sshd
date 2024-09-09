package sshd

import (
	"sync"

	"golang.org/x/crypto/ssh"
)

// DefaultServeMux is the default ServeMux used by Serve.
var DefaultServeMux = NewServeMux()

// Handler 在建立 channel 时调用 ServeChannel.
type Handler interface {
	// ServeChannel 对 ssh.NewChannel 处理, ctx 来源于 ConnContext,
	// 如果返回的 error 不为 nil 将会在 Server 中输出日志.
	ServeChannel(cc *ChannelChain, conn *ssh.ServerConn, newChannel ssh.NewChannel) error
}

// HandlerFunc 函数类型的 Handler
type HandlerFunc func(cc *ChannelChain, conn *ssh.ServerConn, newChannel ssh.NewChannel) error

// ServeChannel implements Handler.ServeChannel
func (h HandlerFunc) ServeChannel(cc *ChannelChain, conn *ssh.ServerConn, newChannel ssh.NewChannel) error {
	return h(cc, conn, newChannel)
}

// ServeMux is an SSH request multiplexer.
type ServeMux struct {
	mut      sync.RWMutex
	handlers map[string]Handler
}

// NewServeMux allocates and returns a new ServeMux.
func NewServeMux() *ServeMux {
	return &ServeMux{
		handlers: make(map[string]Handler),
	}
}

// Handle registers the handler for the given channel type.
// Panics If a handler already existed for channel type.
func (mux *ServeMux) Handle(channelType string, handler Handler) {
	mux.mut.Lock()
	defer mux.mut.Unlock()

	if len(channelType) == 0 {
		panic("mux: invalid channel type")
	}
	if handler == nil {
		panic("mux: nil handler")
	}

	if mux.handlers == nil {
		mux.handlers = make(map[string]Handler)
	}
	if _, existed := mux.handlers[channelType]; existed {
		panic("mux: multiple registrations for " + channelType)
	}
	mux.handlers[channelType] = handler
}

// HandleFunc registers the handler function for the given channel type.
func (mux *ServeMux) HandleFunc(channelType string, handler func(*ChannelChain, *ssh.ServerConn, ssh.NewChannel) error) {
	if handler == nil {
		panic("mux: nil handler")
	}
	mux.Handle(channelType, HandlerFunc(handler))
}

// ServeChannel implements Handler
func (mux *ServeMux) ServeChannel(cc *ChannelChain, conn *ssh.ServerConn, newChannel ssh.NewChannel) error {
	channelType := newChannel.ChannelType()
	mux.mut.RLock()
	handler, ok := mux.handlers[channelType]
	mux.mut.RUnlock()

	if ok && handler != nil {
		return handler.ServeChannel(cc, conn, newChannel)
	}
	return newChannel.Reject(ssh.UnknownChannelType, "unsupported channel type")
}
