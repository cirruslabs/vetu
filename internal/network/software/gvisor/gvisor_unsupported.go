//go:build !linux

package gvisor

import (
	"context"
	"errors"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
	"net"
)

var ErrNotSupported = errors.New("gVisor is not supported on this platform")

type GVisor struct{}

func New(rawSocketFD int, gatewayIP net.IP) (*GVisor, error) {
	return nil, ErrNotSupported
}

func (gvisor *GVisor) Stack() *stack.Stack {
	return nil
}

func (gvisor *GVisor) Run(ctx context.Context) error {
	return ErrNotSupported
}
