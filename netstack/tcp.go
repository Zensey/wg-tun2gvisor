package netstack

import (
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"github.com/zensey/wg-tun2gvisor/services/shaper"

	"gvisor.dev/gvisor/pkg/tcpip/adapters/gonet"
	"gvisor.dev/gvisor/pkg/tcpip/transport/tcp"
	"gvisor.dev/gvisor/pkg/waiter"
)

const tcpWaitTimeout = 50 * time.Second

func (tun *NetTun) acceptTCP(req *tcp.ForwarderRequest) {
	if isPrivateIP(net.IP(req.ID().LocalAddress)) {
		log.Printf("Access to private IPv4 subnet is restricted: %s", req.ID().LocalAddress.String())
		return
	}
	
	localAddress := req.ID().LocalAddress
	log.Printf("TCP> %s: %d", localAddress, req.ID().LocalPort)

	outbound, err := net.Dial("tcp", fmt.Sprintf("%s:%d", localAddress, req.ID().LocalPort))
	if err != nil {
		log.Printf("net.Dial() = %v", err)
		req.Complete(true)
		return
	}

	var wq waiter.Queue
	ep, tcpErr := req.CreateEndpoint(&wq)
	if tcpErr != nil {
		log.Printf("req.CreateEndpoint() = %v", tcpErr)
		req.Complete(false)
		return
	}
	conn := gonet.NewTCPConn(&wq, ep)

	wg := sync.WaitGroup{}
	wg.Add(2)

	go tun.cpy(&wg, outbound, conn, 1) // conn -> outbound
	go tun.cpy(&wg, conn, outbound, 2) // outbound -> conn
	wg.Wait()
}

func (tun *NetTun) cpy(wg *sync.WaitGroup, dst, src net.Conn, i int) {
	defer wg.Done()

	//buf := pool.Get(pool.RelayBufferSize)
	//io.CopyBuffer(dst, src, buf)
	//pool.Put(buf)

	r := shaper.NewReader(src, tun.limiter)
	_, err := io.Copy(dst, r)
	if err != nil {
		log.Printf("copy %v", err)
	}

	// Set a deadline for the ReadOperation so that we don't
	// wait forever for a dst that might not respond on
	// a resonable amount of time.
	dst.SetReadDeadline(time.Now().Add(tcpWaitTimeout))
}
