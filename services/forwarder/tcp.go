package forwarder

import (
	"fmt"
	"io"
	"log"
	"net"
	"sync"

	"gvisor.dev/gvisor/pkg/tcpip/adapters/gonet"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
	"gvisor.dev/gvisor/pkg/tcpip/transport/tcp"
	"gvisor.dev/gvisor/pkg/waiter"
)

// func relay(src, dst net.Conn) {
// 	defer src.Close()
// 	defer dst.Close()
// 	buf := global.BufPool.Get(global.ConnBufSize)
// 	defer global.BufPool.Put(buf)

// 	for {
// 		src.SetDeadline(time.Now().Add(global.TcpStreamTimeout))
// 		nr, err := src.Read(buf)
// 		if err != nil {
// 			return
// 		}

// 		dst.SetDeadline(time.Now().Add(global.TcpStreamTimeout))
// 		if nw, err := dst.Write(buf[:nr]); nw < nr || err != nil {
// 			return
// 		}
// 	}
// }

func TCP(s *stack.Stack /*nat map[tcpip.Address]tcpip.Address, natLock *sync.Mutex*/) *tcp.Forwarder {

	return tcp.NewForwarder(s, 0, 100, func(r *tcp.ForwarderRequest) {
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
		// go io.Copy(outbound, conn)
		// io.Copy(conn, outbound)

		wg := sync.WaitGroup{}
		cpy := func(dst, src net.Conn) {
			defer wg.Done()
			io.Copy(dst, src)
			dst.Close()
		}
		wg.Add(2)
		go cpy(outbound, conn)
		go cpy(conn, outbound)
		wg.Wait()

	})
}
