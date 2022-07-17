package netstack

import (
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"github.com/Dreamacro/clash/common/pool"
	"gvisor.dev/gvisor/pkg/tcpip/adapters/gonet"
	"gvisor.dev/gvisor/pkg/tcpip/transport/tcp"
	"gvisor.dev/gvisor/pkg/waiter"
)

const tcpWaitTimeout = 50 * time.Second

func (tun *NetTun) acceptTCP(r *tcp.ForwarderRequest) {
	localAddress := r.ID().LocalAddress
	log.Printf("TCP> %s: %d", localAddress, r.ID().LocalPort)

	outbound, err := net.Dial("tcp", fmt.Sprintf("%s:%d", localAddress, r.ID().LocalPort))
	if err != nil {
		log.Printf("net.Dial() = %v", err)
		r.Complete(true)
		return
	}

	var wq waiter.Queue
	ep, tcpErr := r.CreateEndpoint(&wq)
	if tcpErr != nil {
		log.Printf("r.CreateEndpoint() = %v", tcpErr)
		r.Complete(false)
		return
	}
	conn := gonet.NewTCPConn(&wq, ep)

	wg := sync.WaitGroup{}
	wg.Add(2)

	go cpy(&wg, outbound, conn, 1) // conn -> outbound
	go cpy(&wg, conn, outbound, 2) // outbound -> conn
	wg.Wait()
}

func cpy(wg *sync.WaitGroup, dst, src net.Conn, i int) {
	defer wg.Done()

	buf := pool.Get(pool.RelayBufferSize)
	io.CopyBuffer(dst, src, buf)
	pool.Put(buf)

	// Set a deadline for the ReadOperation so that we don't
	// wait forever for a dst that might not respond on
	// a resonable amount of time.
	dst.SetReadDeadline(time.Now().Add(tcpWaitTimeout))
}
