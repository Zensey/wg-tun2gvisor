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
	"github.com/zensey/wg-userspace-tun/services/forwarder"
	"github.com/zensey/wg-userspace-tun/services/handler"

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

type netTun struct {
	stack      *stack.Stack
	dispatcher stack.NetworkDispatcher

	events         chan tun.Event
	incomingPacket chan buffer.VectorisedView
	mtu            int
	dnsServers     []netip.Addr

	//
	w *pcapgo.Writer
	f *os.File
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

	tcpForwarder := forwarder.TCP(s)
	s.SetTransportProtocolHandler(tcp.ProtocolNumber, tcpForwarder.HandlePacket)

	icmpHandler := handler.ICMPHandler(s)
	s.SetTransportProtocolHandler(icmp.ProtocolNumber4, icmpHandler)

	dev := &netTun{
		stack:          s,
		events:         make(chan tun.Event, 10),
		incomingPacket: make(chan buffer.VectorisedView),
		dnsServers:     dnsServers,
		mtu:            mtu,
	}

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

	// for _, ip := range localAddresses {
	// 	protoAddr := tcpip.ProtocolAddress{
	// 		Protocol:          ipv4.ProtocolNumber,
	// 		AddressWithPrefix: tcpip.Address(ip.AsSlice()).WithPrefix(),
	// 	}
	// 	log.Println("AddProtocolAddress", protoAddr, header.IPv4EmptySubnet.String())
	// 	tcpipErr := dev.stack.AddProtocolAddress(1, protoAddr, stack.AddressProperties{})
	// 	if tcpipErr != nil {
	// 		return nil, nil, fmt.Errorf("AddProtocolAddress(%v): %v", ip, tcpipErr)
	// 	}
	// }
	// dev.stack.AddRoute(tcpip.Route{Destination: header.IPv4EmptySubnet, NIC: 1})
	// log.Println("dev.stack.GetRouteTable()", dev.stack.GetRouteTable())

	// Enable forwarding
	// s.SetForwardingDefaultAndAllNICs(ipv4.ProtocolNumber, false)

	s.SetRouteTable([]tcpip.Route{
		{
			Destination: header.IPv4EmptySubnet,
			NIC:         1,
		},
	})

	dev.events <- tun.EventUp
	return dev, (*Net)(dev), nil
}

func (tun *netTun) Name() (string, error) {
	return "go", nil
}

func (tun *netTun) File() *os.File {
	return nil
}

func (tun *netTun) Events() chan tun.Event {
	return tun.events
}

func (tun *netTun) Read(buf []byte, offset int) (int, error) {
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

func (tun *netTun) Write(buf []byte, offset int) (int, error) {
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

func (tun *netTun) Flush() error {
	return nil
}

func (tun *netTun) Close() error {
	tun.stack.RemoveNIC(1)

	if tun.events != nil {
		close(tun.events)
	}
	if tun.incomingPacket != nil {
		close(tun.incomingPacket)
	}
	return nil
}

func (tun *netTun) MTU() (int, error) {
	return tun.mtu, nil
}
