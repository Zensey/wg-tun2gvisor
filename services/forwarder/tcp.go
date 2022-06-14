package forwarder

import (
	"context"
	"fmt"
	"log"
	"net"

	"gvisor.dev/gvisor/pkg/tcpip/adapters/gonet"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
	"gvisor.dev/gvisor/pkg/tcpip/transport/tcp"
	"gvisor.dev/gvisor/pkg/waiter"
	"inet.af/tcpproxy"
)

func TCP(s *stack.Stack /*nat map[tcpip.Address]tcpip.Address, natLock *sync.Mutex*/) *tcp.Forwarder {
	return tcp.NewForwarder(s, 0, 10, func(r *tcp.ForwarderRequest) {
		localAddress := r.ID().LocalAddress

		// if linkLocal().Contains(localAddress) {
		// 	r.Complete(true)
		// 	return
		// }
		// natLock.Lock()
		// if replaced, ok := nat[localAddress]; ok {
		// 	localAddress = replaced
		// }
		// natLock.Unlock()

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

		remote := tcpproxy.DialProxy{
			DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
				return outbound, nil
			},
		}
		remote.HandleConn(gonet.NewTCPConn(&wq, ep))
	})
}

// const linkLocalSubnet = "169.254.0.0/16"

// func linkLocal() *tcpip.Subnet {
// 	_, parsedSubnet, _ := net.ParseCIDR(linkLocalSubnet) // CoreOS VM tries to connect to Amazon EC2 metadata service
// 	subnet, _ := tcpip.NewSubnet(tcpip.Address(parsedSubnet.IP), tcpip.AddressMask(parsedSubnet.Mask))
// 	return &subnet
// }
