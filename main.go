package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path"
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

// GREET -> AUTH -> TRANS -> UPDATE
//
// consider maildrop for this project as ~/.pop3/test.com/test for test@test.com
// the directory ~/.pop3/test.com contains test, .hash
// .hash file contains hash of user and pass
// a test user test@test.com with pass pass
const (
	GREET  string = "GREET"
	AUTH   string = "AUTH"
	TRANS  string = "TRANS"
	UPDATE string = "UPDATE"
)

type ClientSession struct {
	State       string
	Conn        net.Conn
	Name        string
	Pass        string
	MailBoxPath string
}

func newClientSession(conn net.Conn) *ClientSession {
	return &ClientSession{
		State: GREET,
		Conn:  conn,
	}
}

func handleCommand(session *ClientSession, cmd string, args []string) {
	cmd = strings.ToUpper(cmd)
	switch cmd {
	case QUIT:
		commandQUIT(session)
	case USER:
		commandUSER(session, args)
	case PASS:
		commandPASS(session)
	}
}

func commandQUIT(session *ClientSession) {
	conn := session.Conn
	conn.Write([]byte("+OK POP3 server signing off\r\n"))
}

// USER username@domain
// mailbox -> /.pop3/domain/username
func commandUSER(session *ClientSession, args []string) {
	if session.State != AUTH {
		session.Conn.Write([]byte("-ERR command not available\r\n"))
		return
	}
	if len(args) == 0 {
		session.Conn.Write([]byte("-ERR argument incomplete\r\n"))
		return
	}
	s := strings.Split(args[0], "@")
	if len(s) < 2 {
		session.Conn.Write([]byte("-ERR argument incomplete\r\n"))
		return
	}
	// name -> s[0], domain -> s[1]
	hdir, err := os.UserHomeDir()
	if err != nil {
		session.Conn.Write([]byte("-ERR mailbox read failed\r\n"))
	}
	mailbox := path.Join(hdir, ".pop3", s[1], s[0])
	// check for the file ~/.pop3/domain/name
	_, err = os.Open(mailbox)
	if os.IsNotExist(err) {
		s := fmt.Sprintf("-ERR never heard of %s\r\n", args[0])
		session.Conn.Write([]byte(s))
		return
	}
	session.MailBoxPath = mailbox
	session.Name = s[0]
}

func commandPASS(session *ClientSession) {
	if session.State != AUTH {
		session.Conn.Write([]byte("-ERR command not available\r\n"))
		return
	}
	if len(session.Name) == 0 {
		session.Conn.Write([]byte("-ERR invalid password\r\n"))
		return
	}
}

func handleConn(session *ClientSession) {
	conn := session.Conn
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(10 * time.Minute))
	conn.Write([]byte("+OK POP3 server ready\r\n"))
	session.State = AUTH
	packets := []byte{}
	tmp := make([]byte, 1024)
	for {
		n, err := conn.Read(tmp)
		if err != nil {
			if err != io.EOF {
				conn.Write([]byte("-ERR read failed\r\n"))
			} else {
				commandQUIT(session)
			}
			return
		}
		packets = append(packets, tmp[:n]...)
		if strings.HasSuffix(string(packets), "\r\n") {
			cmd := string(packets)
			cmd = cmd[:len(cmd)-2]
			packets = []byte{}
			args := strings.Split(cmd, " ")
			handleCommand(session, args[0], args[1:])
		}
	}
}

func main() {
	ln, err := net.Listen("tcp", "0.0.0.0:5000")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("server listening at port 5000")
	for {
		conn, err := ln.Accept()
		if err != nil {
			conn.Write([]byte("-ERR connection failed\r\n"))
			return
		}
		session := newClientSession(conn)
		go handleConn(session)
	}
}
