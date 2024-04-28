package main

import (
	"errors"
	"flag"
	"io"
	"log"
	"net"
	"os"
)

func BindTcp(localAddr string, localPort string, ch chan *net.TCPConn) {
	addr, err := net.ResolveTCPAddr("tcp", "0.0.0.0:"+localPort)
	if err != nil {
		log.Println("Error resolving TCP address: %v", err)
	}

	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		log.Println("Error listening on TCP port: %v", err)
	}
	defer listener.Close()

	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			log.Println("Error accepting TCP connection: %v", err)
			continue
		}
		ch <- conn
	}
}

func ConnectUDP(remoteAddr string, remotePort string) (*net.UDPConn, *net.UDPAddr, error) {
	addr, err := net.ResolveUDPAddr("udp", remoteAddr+":"+remotePort)
	if err != nil {
		return nil, nil, err
	}
	conn, err := net.DialUDP("udp", nil, addr)
	return conn, addr, err
}

func tcpToudp(localAddr string, localPort string, remoteAddr string, remotePort string) {
	tcpconns := make(chan *net.TCPConn)
	go BindTcp(localAddr, localPort, tcpconns)
	for t := range tcpconns {
		u, a, err := ConnectUDP(remoteAddr, remotePort)
		if err != nil {
			continue
		}
		go process(t, u, a)
	}
}

func process(t *net.TCPConn, u *net.UDPConn, udpAddr *net.UDPAddr) {

	defer func() {
		_, _ = u.Write([]byte("#close#"))
		err := t.Close()
		if err != nil {
			return
		}
		err = u.Close()
		if err != nil {
			return
		}
		log.Println("Disconnected with Intranet TCP " + t.RemoteAddr().String())
	}()

	go func() {
		_, err := u.Write([]byte("#start#"))
		if err != nil {
			return
		}
	}()

	log.Println("开始将来自" + t.RemoteAddr().String() + " 的TCP流量封装为UDP并转发。")

	// UDP 数据包最大长度，减去IP和UDP头
	const maxUDPpayload = 1472 // 1500(Ethernet MTU) - 20(IP header) - 8(UDP header)

	go func() {
		// 从UDP到TCP
		buf := make([]byte, maxUDPpayload)
		for {
			n, addr, err := u.ReadFromUDP(buf) // 从UDP读取数据。
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					log.Println(net.ErrClosed)
					return
				}
				log.Println("Error reading from UDP:", err)
				continue
			}

			if addr.String() == udpAddr.String() { // 校验UDP数据包来源地址。

				log.Println(addr.String(), "udp to tcp")
				_, err = t.Write(buf[:n]) // 将读取的数据写入TCP。
				if err != nil {
					if err != io.EOF {
						log.Println("Error writing to TCP:", err)
					}
					continue
				}
			}

		}
	}()

	// 从TCP到UDP
	buf := make([]byte, maxUDPpayload) // 创建一个足够大的缓冲区来读取并发送数据。
	for {
		n, err := t.Read(buf) // 从TCP读取数据。
		if err != nil {
			if err != io.EOF {
				log.Println("Error reading from TCP:", err)
			}
			return
		}

		// 将读取的数据分片发送到UDP
		for offset := 0; offset < n; {
			end := offset + maxUDPpayload
			if end > n {
				end = n
			}
			log.Println(t.RemoteAddr().String(), "tcp to udp")
			_, err = u.Write(buf[offset:end])
			if err != nil {
				log.Println("Error writing to UDP:", err)
				return
			}
			offset = end
		}
	}
}

func main() {
	// Configure logging
	log.SetOutput(os.Stdout)

	var localAddr string
	var localPort string
	var remoteAddr string
	var remotePort string
	flag.StringVar(&localAddr, "la", "", "The local Addr")
	flag.StringVar(&localPort, "lp", "", "The local Port")
	flag.StringVar(&remoteAddr, "ra", "", "The remote Addr")
	flag.StringVar(&remotePort, "rp", "", "The remote Port")
	flag.Parse()

	if localAddr == "" || localPort == "" || remoteAddr == "" || remotePort == "" {
		flag.Usage()
		os.Exit(1)
	}

	tcpToudp(localAddr, localPort, remoteAddr, remotePort)
}
