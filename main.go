package main

import (
	"log"
	"net"
	"strings"
)

func handleConn(conn net.Conn, err error) {
	if err != nil {
		conn.Write([]byte("-ERR connection falied\r\n"))
		conn.Close()
		return
	}
	conn.Write([]byte("+OK POP3 server ready\r\n"))
	packets := make([]byte, 1024)
	tmp := make([]byte, 1024)
	for {
		_, err := conn.Read(tmp)
		if err != nil {
			conn.Write([]byte("-ERR connection failed\r\n"))
			conn.Close()
			return
		}
		packets = append(packets, tmp...)
		if strings.Contains(string(packets), "QUIT") {
			conn.Write([]byte("+OK POP3 server signing off\r\n"))
			conn.Close()
			return
		}
		packets = make([]byte, 1024)
	}
}

func main() {
	ln, err := net.Listen("tcp", ":5000")
	if err != nil {
		log.Fatal(err)
	}
	for {
		conn, err := ln.Accept()
		go handleConn(conn, err)
	}
}
