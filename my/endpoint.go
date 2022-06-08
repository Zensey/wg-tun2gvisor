package my

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcapgo"
	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
)

// implements PacketEndpoint interface

type MyPacketEndpoint struct {
	w *pcapgo.Writer
	f *os.File
}

func NewMyPacketEndpoint() *MyPacketEndpoint {
	fmt.Println("hp> NewMyPacketEndpoint >>>>>>>>>>>>>")
	d := &MyPacketEndpoint{}

	d.f, _ = os.Create("file.pcap")
	d.w = pcapgo.NewWriterNanos(d.f)
	d.w.WriteFileHeader(65536, layers.LinkTypeEthernet) // new file, must do this.

	return d
}

func (a *MyPacketEndpoint) HandlePacket(nicID tcpip.NICID, addr tcpip.LinkAddress, netProto tcpip.NetworkProtocolNumber, pkt *stack.PacketBuffer) {
	fmt.Println("hp> HandlePacket >", nicID, addr)

	var pkts stack.PacketBufferList
	pkts.PushBack(pkt)
	log.Println("tun> DeliverNetworkPacket", pkts)

	for pkt := pkts.Front(); pkt != nil; pkt = pkt.Next() {
		// log.Println("tun> Write", buffer.NewVectorisedView(pkt.Size(), pkt.Views()))
		log.Println("tun> Write >", pkt.Data().AsRange().AsView())

		p := make([]byte, 0)
		p = append(p, []byte{0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 8, 0}...)
		p = append(p, (pkt.Data().AsRange().AsView())...)
		err := a.w.WritePacket(gopacket.CaptureInfo{Timestamp: time.Now(), CaptureLength: len(p), Length: len(p)}, p)
		if err != nil {
			log.Println("tun> Write", err)
		}

		// err := a.w.WritePacket(gopacket.CaptureInfo{Timestamp: time.Now(), CaptureLength: pkt.Size(), Length: pkt.Size()}, pkt.Data().AsRange().AsView())
		// if err != nil {
		// 	log.Println("tun> Write", err)
		// }
	}

}
