package main

import (
	"io"
	"log"
	"net"
	"strings"
	"time"
)

const (
	QUIT string = "QUIT"
	STAT string = "STAT"
	LIST string = "LIST"
	RETR string = "RETR"
	DELE string = "DELE"
	NOOP string = "NOOP"
	RSET string = "RSET"
	TOP  string = "TOP"
	UIDL string = "UIDL"
	USER string = "USER"
	PASS string = "PASS"
)

func handleCommand(conn net.Conn, cmd string, args []string) {
	cmd = strings.ToUpper(cmd)
	switch cmd {
	case QUIT:
		commandQUIT(conn)
	}
}

func commandQUIT(conn net.Conn) {
	conn.Write([]byte("+OK POP3 server signing off\r\n"))
}

func handleConn(conn net.Conn, err error) {
	if err != nil {
		conn.Write([]byte("-ERR connection falied\r\n"))
		return
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(10 * time.Minute))
	conn.Write([]byte("+OK POP3 server ready\r\n"))
	packets := []byte{}
	tmp := make([]byte, 1024)
	for {
		n, err := conn.Read(tmp)
		if err != nil {
			if err != io.EOF {
				conn.Write([]byte("-ERR read failed\r\n"))
			} else {
				commandQUIT(conn)
			}
			return
		}
		packets = append(packets, tmp[:n]...)
		if strings.HasSuffix(string(packets), "\r\n") {
			cmd := string(packets)
			cmd = cmd[:len(cmd)-2]
			packets = []byte{}
			args := strings.Split(cmd, " ")
			handleCommand(conn, args[0], args[1:])
		}
	}
}

func main() {
	ln, err := net.Listen("tcp", "0.0.0.0:5000")
	if err != nil {
		log.Fatal(err)
	}
	for {
		conn, err := ln.Accept()
		go handleConn(conn, err)
	}
}
