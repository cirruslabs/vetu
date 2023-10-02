package gvisor

import (
	"context"
	"errors"
	"fmt"
	"github.com/cirruslabs/vetu/internal/randommac"
	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/adapters/gonet"
	"gvisor.dev/gvisor/pkg/tcpip/header"
	"gvisor.dev/gvisor/pkg/tcpip/link/fdbased"
	"gvisor.dev/gvisor/pkg/tcpip/network/arp"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv4"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
	"gvisor.dev/gvisor/pkg/tcpip/transport/icmp"
	"gvisor.dev/gvisor/pkg/tcpip/transport/tcp"
	"gvisor.dev/gvisor/pkg/tcpip/transport/udp"
	"gvisor.dev/gvisor/pkg/waiter"
	"io"
	"net"
)

const nicID = 1

var ErrInitFailed = errors.New("failed to initialize gVisor")

type gVisorHandler func(stack.TransportEndpointID, stack.PacketBufferPtr) bool

type GVisor struct {
	st *stack.Stack
}

func New(rawSocketFD int, gatewayIP net.IP) (*GVisor, error) {
	// Create network stack
	st := stack.New(stack.Options{
		NetworkProtocols: []stack.NetworkProtocolFactory{
			arp.NewProtocol,
			ipv4.NewProtocol,
		},
		TransportProtocols: []stack.TransportProtocolFactory{
			tcp.NewProtocol,
			udp.NewProtocol,
			icmp.NewProtocol4,
		},
	})

	// Create network interface
	macAddress, err := randommac.UnicastAndLocallyAdministered()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to generate a random MAC-address: %v",
			ErrInitFailed, err)
	}

	linkEndpoint, err := fdbased.New(&fdbased.Options{
		FDs:                []int{rawSocketFD},
		MTU:                1500,
		EthernetHeader:     true,
		Address:            tcpip.LinkAddress(macAddress),
		PacketDispatchMode: fdbased.PacketMMap,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: failed to create an FD-based endpoint: %v",
			ErrInitFailed, err)
	}

	if err := st.CreateNIC(nicID, linkEndpoint); err != nil {
		return nil, fmt.Errorf("%w: failed to create NIC: %v",
			ErrInitFailed, err)
	}

	// Switch interface into promiscuous mode to capture packets
	// not directly destined to us, thus enabling the gateway
	// functionality
	if err := st.SetPromiscuousMode(nicID, true); err != nil {
		return nil, fmt.Errorf("%w: failed to set promiscuous mode on a NIC: %v",
			ErrInitFailed, err)
	}

	// Enable spoofing on the interface to allow replying from addresses
	// other than set on the interface, thus enabling the gateway
	// functionality
	if err := st.SetSpoofing(nicID, true); err != nil {
		return nil, fmt.Errorf("%w: failed to enable spoofing on a NIC: %v",
			ErrInitFailed, err)
	}

	// Set interface address and add a route
	if err := st.AddProtocolAddress(nicID, tcpip.ProtocolAddress{
		Protocol: ipv4.ProtocolNumber,
		AddressWithPrefix: tcpip.AddressWithPrefix{
			Address:   tcpip.AddrFrom4Slice(gatewayIP.To4()),
			PrefixLen: 29,
		},
	}, stack.AddressProperties{}); err != nil {
		return nil, fmt.Errorf("%w: failed to add IPv4 address: %v",
			ErrInitFailed, err)
	}

	st.AddRoute(tcpip.Route{Destination: header.IPv4EmptySubnet, NIC: nicID})

	// Pre-create the gVisor structure, otherwise we won't be able
	// to reference the TCP and UDP forwarder handlers
	gvisor := &GVisor{
		st: st,
	}

	// Configure TCP forwarder
	tcpForwarder := tcp.NewForwarder(st, 0, 1000, gvisor.forwardTCP)
	st.SetTransportProtocolHandler(tcp.ProtocolNumber, withForwardingFilter(tcpForwarder.HandlePacket))

	// Configure UDP forwarder
	udpForwarder := udp.NewForwarder(st, gvisor.forwardUDP)
	st.SetTransportProtocolHandler(udp.ProtocolNumber, withForwardingFilter(udpForwarder.HandlePacket))

	return gvisor, nil
}

func (gvisor *GVisor) Stack() *stack.Stack {
	return gvisor.st
}

func (gvisor *GVisor) Run(ctx context.Context) error {
	go func() {
		<-ctx.Done()
		gvisor.st.Close()
	}()

	gvisor.st.Wait()

	return nil
}

func (gvisor *GVisor) forwardTCP(request *tcp.ForwarderRequest) {
	var wq waiter.Queue

	ep, tcpipErr := request.CreateEndpoint(&wq)
	if tcpipErr != nil {
		fmt.Printf("failed to create TCP endpoint: %v\n", tcpipErr)

		request.Complete(true)
		return
	}

	guestConn := gonet.NewTCPConn(&wq, ep)

	remoteConn, err := net.Dial("tcp", fmt.Sprintf("%s:%d",
		request.ID().LocalAddress.String(), request.ID().LocalPort))
	if err != nil {
		request.Complete(true)

		return
	}

	request.Complete(false)

	go func() {
		go func() {
			_, _ = io.Copy(remoteConn, guestConn)
		}()

		_, _ = io.Copy(guestConn, remoteConn)
	}()
}

func (gvisor *GVisor) forwardUDP(request *udp.ForwarderRequest) {
	var wq waiter.Queue

	ep, tcpipErr := request.CreateEndpoint(&wq)
	if tcpipErr != nil {
		fmt.Printf("failed to create UDP endpoint: %v\n", tcpipErr)

		return
	}

	guestConn := gonet.NewUDPConn(gvisor.st, &wq, ep)

	remoteConn, err := net.Dial("udp", fmt.Sprintf("%s:%d",
		request.ID().LocalAddress.String(), request.ID().LocalPort))
	if err != nil {
		return
	}

	go func() {
		go func() {
			_, _ = io.Copy(remoteConn, guestConn)
		}()

		_, _ = io.Copy(guestConn, remoteConn)
	}()
}

func withForwardingFilter(h gVisorHandler) gVisorHandler {
	return func(id stack.TransportEndpointID, ptr stack.PacketBufferPtr) bool {
		// A "local" address presented to us as a gateway
		// is actually a remote address that the guest wants
		// to connect/send packets to
		remoteAddress := id.LocalAddress

		// Only forward IPv4 traffic
		if remoteAddress.Len() != 4 {
			return false
		}

		// Convert to net.IP
		remoteIPRaw := remoteAddress.As4()
		remoteIP := net.IPv4(remoteIPRaw[0], remoteIPRaw[1], remoteIPRaw[2], remoteIPRaw[3])

		// Only forward global unicast addresses, including private IPv4 address space
		if !remoteIP.IsGlobalUnicast() {
			return false
		}

		return h(id, ptr)
	}
}
