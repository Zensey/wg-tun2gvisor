package netstack

import (
	"context"
	"log"
	"net"
	"time"

	"gvisor.dev/gvisor/pkg/tcpip/adapters/gonet"
	"gvisor.dev/gvisor/pkg/tcpip/transport/udp"
	"gvisor.dev/gvisor/pkg/waiter"
)

const (
	idleTimeout = 10 * time.Second
)

func (tun *NetTun) acceptUDP(req *udp.ForwarderRequest) {
	sess := req.ID()
	log.Println("acceptUDP>", sess.LocalAddress, sess.RemoteAddress)

	var wq waiter.Queue

	ep, udpErr := req.CreateEndpoint(&wq)
	if udpErr != nil {
		log.Printf("udpErr %v", udpErr)
		return
	}
	client := gonet.NewUDPConn(tun.stack, &wq, ep)

	clientAddr := &net.UDPAddr{IP: net.IP([]byte(sess.RemoteAddress)), Port: int(sess.RemotePort)}
	remoteAddr := &net.UDPAddr{IP: net.IP([]byte(sess.LocalAddress)), Port: int(sess.LocalPort)}
	proxyAddr := &net.UDPAddr{IP: net.ParseIP("0.0.0.0"), Port: int(sess.RemotePort)}

	if remoteAddr.Port == 53 && tun.isLocal(sess.LocalAddress) {
		remoteAddr.Port = 53
		remoteAddr.IP = net.ParseIP("127.0.0.1")
	}

	proxyConn, err := net.ListenUDP("udp", proxyAddr)
	if err != nil {
		log.Printf("Failed to bind local port %d, trying one more time with random port", proxyAddr)
		proxyAddr.Port = 0

		proxyConn, err = net.ListenUDP("udp", proxyAddr)
		if err != nil {
			log.Printf("Failed to bind local random port %s", proxyAddr)
			return
		}
	}
	ctx, cancel := context.WithCancel(context.Background())

	go tun.proxy(ctx, cancel, client, clientAddr, proxyConn)
	go tun.proxy(ctx, cancel, proxyConn, remoteAddr, client)
}

func (tun *NetTun) proxy(ctx context.Context, cancel context.CancelFunc, dst net.PacketConn, dstAddr net.Addr, src net.PacketConn) {
	defer cancel()
	buf := make([]byte, tun.mtu)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			src.SetReadDeadline(time.Now().Add(idleTimeout))
			n, srcAddr, err := src.ReadFrom(buf)
			if e, ok := err.(net.Error); ok && e.Timeout() {
				return
			} else if err != nil {
				if ctx.Err() == nil {
					log.Printf("Failed to read packed from %s", srcAddr)
				}
				return
			}

			_, err = dst.WriteTo(buf[:n], dstAddr)
			if err != nil {
				if ctx.Err() == nil {
					log.Printf("Failed to write packed to %s", dstAddr)
				}
				return
			}
			dst.SetReadDeadline(time.Now().Add(idleTimeout))
		}
	}
}
