package netstack

import (
	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/buffer"
	"gvisor.dev/gvisor/pkg/tcpip/header"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
)

type (
	endpoint NetTun
	Net      NetTun
)

func (e *endpoint) Attach(dispatcher stack.NetworkDispatcher) {
	e.dispatcher = dispatcher
}

func (e *endpoint) IsAttached() bool {
	return e.dispatcher != nil
}

func (e *endpoint) MTU() uint32 {
	mtu, err := (*NetTun)(e).MTU()
	if err != nil {
		panic(err)
	}
	return uint32(mtu)
}

func (*endpoint) Capabilities() stack.LinkEndpointCapabilities {
	return stack.CapabilityNone
}

func (*endpoint) MaxHeaderLength() uint16 {
	return 0
}

func (*endpoint) LinkAddress() tcpip.LinkAddress {
	return ""
}

func (*endpoint) Wait() {}

func (e *endpoint) WritePacket(_ stack.RouteInfo, _ tcpip.NetworkProtocolNumber, pkt *stack.PacketBuffer) tcpip.Error {
	// log.Println("endpoint > WritePacket", pkt)

	e.incomingPacket <- buffer.NewVectorisedView(pkt.Size(), pkt.Views())
	return nil
}

// func (e *endpoint) WritePackets(stack.RouteInfo, stack.PacketBufferList, tcpip.NetworkProtocolNumber) (int, tcpip.Error) {
// 	panic("not implemented")
// }

func (e *endpoint) WritePackets(l stack.PacketBufferList) (int, tcpip.Error) {
	// panic("not implemented")
	// log.Println("WritePackets>", l)

	for _, pkt := range l.AsSlice() {
		e.incomingPacket <- buffer.NewVectorisedView(pkt.Size(), pkt.Views())
	}

	// for pkt := l.Front(); pkt != nil; pkt = pkt.Next() {
	// 	e.incomingPacket <- buffer.NewVectorisedView(pkt.Size(), pkt.Views())
	// }
	return 0, nil
}

func (e *endpoint) WriteRawPacket(*stack.PacketBuffer) tcpip.Error {
	panic("not implemented")
}

func (*endpoint) ARPHardwareType() header.ARPHardwareType {
	return header.ARPHardwareNone
}

// func (e *endpoint) AddHeader(tcpip.LinkAddress, tcpip.LinkAddress, tcpip.NetworkProtocolNumber, *stack.PacketBuffer) {}
func (e *endpoint) AddHeader(*stack.PacketBuffer) {}
