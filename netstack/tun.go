/* SPDX-License-Identifier: MIT
 *
 * Copyright (C) 2019-2021 WireGuard LLC. All Rights Reserved.
 */

package netstack

import (
	"fmt"
	"log"
	"net/netip"
	"os"
	"time"

	"golang.zx2c4.com/wireguard/tun"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcapgo"
	"github.com/zensey/wg-tun2gvisor/services/handler"

	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/buffer"
	"gvisor.dev/gvisor/pkg/tcpip/header"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv4"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv6"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
	"gvisor.dev/gvisor/pkg/tcpip/transport/icmp"
	"gvisor.dev/gvisor/pkg/tcpip/transport/tcp"
	"gvisor.dev/gvisor/pkg/tcpip/transport/udp"
)

type NetTun struct {
	stack      *stack.Stack
	dispatcher stack.NetworkDispatcher

	events         chan tun.Event
	incomingPacket chan buffer.VectorisedView
	mtu            int

	localAddresses []netip.Addr
	dnsServers     []netip.Addr

	//
	closed bool
	w      *pcapgo.Writer
	f      *os.File
}

func CreateNetTUN(localAddresses, dnsServers []netip.Addr, mtu int) (tun.Device, *Net, error) {
	fmt.Println("tun>", localAddresses)

	s := stack.New(stack.Options{
		NetworkProtocols: []stack.NetworkProtocolFactory{
			ipv4.NewProtocol,
		},
		TransportProtocols: []stack.TransportProtocolFactory{
			tcp.NewProtocol,
			udp.NewProtocol,
			icmp.NewProtocol4,
		},
		// AllowPacketEndpointWrite: true,
		// HandleLocal:              false,
	})
	dev := &NetTun{
		stack:          s,
		events:         make(chan tun.Event, 10),
		incomingPacket: make(chan buffer.VectorisedView),
		dnsServers:     dnsServers,
		mtu:            mtu,
		localAddresses: localAddresses,
	}

	s.SetTransportProtocolHandler(tcp.ProtocolNumber, tcp.NewForwarder(s, 0, 10000, dev.acceptTCP).HandlePacket)
	s.SetTransportProtocolHandler(udp.ProtocolNumber, udp.NewForwarder(s, dev.acceptUDP).HandlePacket)

	icmpHandler := handler.ICMPHandler(s)
	s.SetTransportProtocolHandler(icmp.ProtocolNumber4, icmpHandler)

	if false {
		dev.f, _ = os.Create("file.pcap")
		dev.w = pcapgo.NewWriterNanos(dev.f)
		dev.w.WriteFileHeader(65536, layers.LinkTypeEthernet) // new file, must do this.
	}

	tcpipErr := s.CreateNIC(1, (*endpoint)(dev) /*, myEP*/)
	if tcpipErr != nil {
		return nil, nil, fmt.Errorf("CreateNIC: %v", tcpipErr)
	}

	// forwarding
	// s.SetNICForwarding(1, ipv4.ProtocolNumber, false)

	s.SetSpoofing(1, true)
	s.SetPromiscuousMode(1, true)

	// dns
	if err := dnsServer(s); err != nil {
		return nil, nil, err
	}

	s.SetRouteTable([]tcpip.Route{
		{
			Destination: header.IPv4EmptySubnet,
			NIC:         1,
		},
	})

	dev.events <- tun.EventUp
	return dev, (*Net)(dev), nil
}

func (tun *NetTun) Name() (string, error) {
	return "go", nil
}

func (tun *NetTun) File() *os.File {
	return nil
}

func (tun *NetTun) Events() chan tun.Event {
	return tun.events
}

func (tun *NetTun) Read(buf []byte, offset int) (int, error) {
	// log.Println("tun> Read")

	view, ok := <-tun.incomingPacket
	if !ok {
		return 0, os.ErrClosed
	}
	n, err := view.Read(buf[offset:])
	if false && n > 0 {
		p := make([]byte, 0)
		p = append(p, []byte{0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 8, 0}...)
		p = append(p, buf[offset:offset+n]...)
		err := tun.w.WritePacket(gopacket.CaptureInfo{Timestamp: time.Now(), CaptureLength: len(p), Length: len(p)}, p)
		if err != nil {
			log.Println("tun> Write", err)
		}
	}

	return n, err
}

func (tun *NetTun) Write(buf []byte, offset int) (int, error) {
	// log.Println("tun> Write")

	packet := buf[offset:]
	if len(packet) == 0 {
		return 0, nil
	}

	pkb := stack.NewPacketBuffer(stack.PacketBufferOptions{Data: buffer.NewVectorisedView(len(packet), []buffer.View{buffer.NewViewFromBytes(packet)})})
	switch packet[0] >> 4 {
	case 4:
		// log.Println("tun>>> DeliverNetworkPacket 4>")
		tun.dispatcher.DeliverNetworkPacket(ipv4.ProtocolNumber, pkb)
	case 6:
		// log.Println("tun>>> DeliverNetworkPacket 6>")
		tun.dispatcher.DeliverNetworkPacket(ipv6.ProtocolNumber, pkb)
	}
	pkb.DecRef()

	return len(buf), nil
}

func (tun *NetTun) Flush() error {
	return nil
}

func (tun *NetTun) Close() error {
	log.Println("tun > Close")

	if !tun.closed {
		if tun.events != nil {
			close(tun.events)
		}
		if tun.incomingPacket != nil {
			close(tun.incomingPacket)
		}
		tun.closed = true
		tun.stack.RemoveNIC(1)
	}
	return nil
}

func (tun *NetTun) MTU() (int, error) {
	return tun.mtu, nil
}

func (tun *NetTun) isLocal(remoteAddr tcpip.Address) bool {
	for _, ip := range tun.localAddresses {
		if tcpip.Address(ip.AsSlice()) == remoteAddr {
			return true
		}
	}

	return false
}