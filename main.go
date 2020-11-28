package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"github.com/txthinking/socks5"
	"golang.org/x/net/proxy"
	"net"
	"os"
)

func main() {
	var flags struct {
		Bind        string
		SocksServer string
		SocksUDP    bool
		DNSServer   string
	}
	flag.StringVar(&flags.Bind, "bind", "127.0.0.1:8053", "address to bind")
	flag.StringVar(&flags.SocksServer, "socks-server", "127.0.0.1:1080", "socks5 server address")
	flag.BoolVar(&flags.SocksUDP, "socks-udp", false, "dial socks5 using udp")
	flag.StringVar(&flags.DNSServer, "dns-server", "8.8.8.8:53", "remote dns server")
	flag.Parse()
	network := "tcp"
	if flags.SocksUDP {
		network = "udp"
	}
	dialer, _ := socks5.NewClient(flags.SocksServer, "", "", 30, 30)
	localConn, err := net.ListenPacket("udp", flags.Bind)
	if err != nil {
		fmt.Fprintln(os.Stderr, "can't bind:", flags.Bind, err)
		os.Exit(1)
	}
	for {
		buf := make([]byte, 64*1024)
		n, addr, _ := localConn.ReadFrom(buf)
		go handleQuery(flags.DNSServer, dialer, network, localConn, n, addr, buf[:n])
	}
}

func handleQuery(dnsServer string, dialer proxy.Dialer, network string, localConn net.PacketConn, n int, addr net.Addr, buf []byte) {
	conn, err := dialer.Dial(network, dnsServer)
	if err != nil {
		fmt.Fprintln(os.Stderr, "connect dns server error:", err)
		return
	}
	defer conn.Close()
	lenBuf := make([]byte, 2)
	if network == "tcp" {
		buf = append(buf, lenBuf...) // just extend the length
		binary.BigEndian.PutUint16(lenBuf, uint16(n))
		copy(buf[2:], buf)
		copy(buf, lenBuf)
	}
	_, err = conn.Write(buf)
	if err != nil {
		fmt.Fprintln(os.Stderr, "write to dns server error:", err)
		return
	}
	var size uint16
	if network == "tcp" {
		n, err = conn.Read(lenBuf)
		if err != nil {
			fmt.Fprintln(os.Stderr, "read length from dns server error:", err)
			return
		}
		size = binary.BigEndian.Uint16(lenBuf)
	}
	n, err = conn.Read(buf)
	if err != nil {
		fmt.Fprintln(os.Stderr, "read data from dns server error:", err)
		return
	}
	if size != 0 && n > int(size) {
		n = int(size)
	}
	n, err = localConn.WriteTo(buf[:n], addr)
	if err != nil {
		fmt.Fprintln(os.Stderr, "back write error:", err.Error())
		return
	}
}
