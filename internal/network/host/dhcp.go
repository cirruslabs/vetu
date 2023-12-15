package host

import (
	"context"
	"errors"
	"fmt"
	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/insomniacslk/dhcp/dhcpv4/server4"
	"net"
	"time"
)

var ErrInitFailed = errors.New("failed to initialize DHCP server")

type DHCP struct {
	gatewayIP net.IP
	vmIP      net.IP

	server *server4.Server
}

func NewDHCPServer(ifname string, gatewayIP net.IP, vmIP net.IP) (*DHCP, error) {
	dhcp := &DHCP{
		gatewayIP: gatewayIP,
		vmIP:      vmIP,
	}

	server, err := server4.NewServer(ifname, nil, dhcp.handle)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInitFailed, err)
	}
	dhcp.server = server

	return dhcp, nil
}

func (dhcp *DHCP) Run(ctx context.Context) error {
	go func() {
		<-ctx.Done()
		dhcp.server.Close()
	}()

	return dhcp.server.Serve()
}

func (dhcp *DHCP) handle(conn net.PacketConn, peer net.Addr, request *dhcpv4.DHCPv4) {
	var messageType dhcpv4.MessageType

	switch request.MessageType() {
	case dhcpv4.MessageTypeDiscover:
		messageType = dhcpv4.MessageTypeOffer
	case dhcpv4.MessageTypeRequest:
		messageType = dhcpv4.MessageTypeAck
	default:
		return
	}

	reply, err := dhcpv4.NewReplyFromRequest(request,
		dhcpv4.WithMessageType(messageType),
		dhcpv4.WithYourIP(dhcp.vmIP),
		dhcpv4.WithOption(dhcpv4.OptSubnetMask(net.CIDRMask(29, 32))),
		dhcpv4.WithRouter(dhcp.gatewayIP),
		dhcpv4.WithDNS(net.ParseIP("8.8.8.8"), net.ParseIP("8.8.4.4")),
		dhcpv4.WithOption(dhcpv4.OptIPAddressLeaseTime(10*time.Minute)),
		dhcpv4.WithOption(dhcpv4.OptServerIdentifier(dhcp.gatewayIP)),
	)
	if err != nil {
		return
	}

	_, err = conn.WriteTo(reply.ToBytes(), peer)
	if err != nil {
		return
	}
}
