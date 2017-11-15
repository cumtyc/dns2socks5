package main

import (
	"golang.org/x/net/proxy"
	"fmt"
	"os"
	"net"
	"encoding/binary"
	"flag"
)


func main(){
	var flags struct {
		Bind        string
		SocksServer string
		DNSServer	string
	}
	flag.StringVar(&flags.Bind, "bind", "127.0.0.1:8053", "address to bind")
	flag.StringVar(&flags.SocksServer, "socks-server", "127.0.0.1:1080", "socks5 server address")
	flag.StringVar(&flags.DNSServer, "dns-server", "8.8.8.8:53", "remote dns server")
	flag.Parse()
	dialer, _ := proxy.SOCKS5("tcp", flags.SocksServer, nil, proxy.Direct)
	localConn, err := net.ListenPacket("udp", flags.Bind)
	if err != nil {
		fmt.Fprintln(os.Stderr, "can't bind:", flags.Bind, err)
		os.Exit(1)
	}
	for{
		buf := make([]byte, 64*1024)
		n, addr, _ := localConn.ReadFrom(buf[2:])
		go handleQuery(flags.DNSServer, dialer, localConn, n, addr, buf)
	}
}


func handleQuery(dnsServer string, dialer proxy.Dialer, localConn net.PacketConn, n int, addr net.Addr, buf []byte){
	conn, err := dialer.Dial("tcp", dnsServer)
	if err != nil {
		fmt.Fprintln(os.Stderr, "connect dns server error:", err)
		return
	}
	defer conn.Close()
	binary.BigEndian.PutUint16(buf, uint16(n))
	_, err = conn.Write(buf[:n+2])
	if err != nil {
		fmt.Fprintln(os.Stderr,"write to dns server error:", err)
		return
	}
	n, err = conn.Read(buf[:2])
	if err != nil {
		fmt.Fprintln(os.Stderr,"read length from dns server error:", err)
		return
	}
	size := binary.BigEndian.Uint16(buf[:2])
	resBuf := buf[2:2+size]
	n, err = conn.Read(resBuf)
	if err != nil {
		fmt.Fprintln(os.Stderr,"read data from dns server error:", err)
		return
	}
	n, err = localConn.WriteTo(resBuf, addr)
	if err != nil {
		fmt.Fprintln(os.Stderr,"back write error %s", err.Error())
		return
	}
}