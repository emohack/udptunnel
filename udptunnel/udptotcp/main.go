package main

import (
	"errors"
	"flag"
	"io"
	"log"
	"net"
	"os"
)

const maxUDPpayload = 1472 // 1500(Ethernet MTU) - 20(IP header) - 8(UDP header)

func BindUDP(localAddr string, localPort string) (*net.UDPConn, error) {

	addr, err := net.ResolveUDPAddr("udp", "0.0.0.0:"+localPort)
	if err != nil {
		log.Println("Error resolving TCP address: %v", err)
	}
	conn, err := net.ListenUDP("udp", addr)
	return conn, err

}

func ConnectTCP(remoteAddr string, remotePort string) (*net.TCPConn, error) {
	addr, err := net.ResolveTCPAddr("tcp", remoteAddr+":"+remotePort)
	if err != nil {
		return nil, err
	}

	conn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		log.Println(err)
	}
	return conn, err
}

func process(localAddr string, localPort string, remoteAddr string, remotePort string) {
	log.Println("start......")
	udpconn, err := BindUDP(localAddr, localPort)
	if err != nil {
		log.Println(err)
		return
	}

	log.Println("监听udp端口", localPort)
	if udpconn == nil {
		return
	}
	dicT := make(map[string]*net.TCPConn)
	dicU := make(map[*net.TCPConn]*net.UDPAddr)
	ch := make(chan *net.TCPConn)

	// udp到tcp
	go func() {
		buf := make([]byte, maxUDPpayload)

		for {
			n, addr, err := udpconn.ReadFromUDP(buf) // 从UDP读取数据。
			if err != nil {
				log.Println(err)
				continue
			}

			data := string(buf[:n])
			if data == "#close#" {
				log.Println(addr.String(), " Closed...")
				v, ok := dicT[addr.String()]
				if ok {
					err := v.Close()
					if err != nil {
						return
					}
					delete(dicU, v)
					delete(dicT, addr.String())
				} else {

				}
				continue
			}

			if _, ok := dicT[addr.String()]; ok {

			} else {
				t, err := ConnectTCP(remoteAddr, remotePort)
				log.Println("成功连接远程端口")
				if err != nil {
					log.Println("连接远端TCP失败")
					continue
				}
				dicT[addr.String()] = t
				dicU[t] = addr
				ch <- t
			}

			t := dicT[addr.String()]

			// 将接受到的udp数据转发到tcp
			if data != "#start#" {
				_, err = t.Write(buf[:n]) // 将读取的数据写入TCP。
				log.Println("udp to tcp", t.RemoteAddr().String())
				if err != nil {
					if errors.Is(err, net.ErrClosed) {
						delete(dicT, addr.String())
					} else if err != io.EOF {
						log.Println("Error writing to TCP:", err)
					}
					continue
				}
			}
		}
	}()

	// 将tcp流量转发回远程udp端口
	for t := range ch {
		go tcpToudp(t, dicU, udpconn)
	}
}

func tcpToudp(t *net.TCPConn, dicU map[*net.TCPConn]*net.UDPAddr, udpconn *net.UDPConn) {
	defer func() {
		delete(dicU, t)
		err := t.Close()
		if err != nil {
			return
		}
	}()

	//接收tcp信息
	buf := make([]byte, maxUDPpayload)
	for {
		n, err := t.Read(buf[:])
		if err != nil {
			return
		}
		// 发送udp消息
		addr := dicU[t]
		log.Println(addr.String(), "tcp to udp")
		_, err = udpconn.WriteToUDP(buf[:n], addr)
		if err != nil {
			log.Println(err)
			return
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

	process(localAddr, localPort, remoteAddr, remotePort)
}
