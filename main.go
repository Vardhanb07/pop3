package main

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
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
// consider maildrop for this project as ~/.pop3/test.com/test.db for test@test.com
// the directory ~/.pop3/test.com contains test, .test_hash
// .hash file contains hash of user and pass
// a test user test@test.com with pass pass
const (
	GREET  string = "GREET"
	AUTH   string = "AUTH"
	TRANS  string = "TRANS"
	UPDATE string = "UPDATE"
)

type ClientSession struct {
	State   string
	Conn    net.Conn
	Name    string
	Pass    string
	Mailbox mailbox
}

type mailbox struct {
	Path string
	Mu   sync.RWMutex
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
		commandPASS(session, args)
	case STAT:
		commandSTAT(session)
	case LIST:
		commandLIST(session, args)
	case RETR:
		commandRETR(session, args)
	case DELE:
		commandDELE(session, args)
	case NOOP:
		commandNOOP(session)
	case RSET:
		commandRSET(session)
	case TOP:
		commandTOP(session, args)
	case UIDL:
		commandUIDL(session, args)
	default:
		session.Conn.Write([]byte("-ERR no such command available\r\n"))
	}
}

func commandQUIT(session *ClientSession) {
	conn := session.Conn
	conn.Write([]byte("+OK POP3 server signing off\r\n"))
}

// USER username@domain
// mailbox -> /.pop3/domain/username.db
func commandUSER(session *ClientSession, args []string) {
	if session.State != AUTH {
		session.Conn.Write([]byte("-ERR action not premitted\r\n"))
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
		return
	}
	mailbox := path.Join(hdir, ".pop3", s[1], s[0]+".db")
	// check for the file ~/.pop3/domain/name.db
	_, err = os.Open(mailbox)
	if os.IsNotExist(err) {
		session.Conn.Write(fmt.Appendf([]byte{}, "-ERR never heard of %s\r\n", args[0]))
		return
	}
	session.Conn.Write(fmt.Appendf([]byte{}, "+OK %s is a valid mailbox\r\n", args[0]))
	session.Mailbox.Path = path.Join(hdir, ".pop3", s[1])
	session.Name = s[0]
}

// PASS pass
// here password with spaces is not handled
// check the password with the hash ~/.pop3/test.com/.test_hash considering test@test.com
func commandPASS(session *ClientSession, args []string) {
	if session.State != AUTH {
		session.Conn.Write([]byte("-ERR action not premitted\r\n"))
		return
	}
	if len(session.Name) == 0 {
		session.Conn.Write([]byte("-ERR invalid password\r\n"))
		return
	}
	if len(args) < 1 {
		session.Conn.Write([]byte("-ERR argument incomplete\r\n"))
		return
	}
	path := path.Join(session.Mailbox.Path, "."+session.Name+"_hash")
	// check for the file ~/.pop3/test.com/.test_hash
	hashfile, _ := os.Open(path)
	hash := make([]byte, 1024)
	n, _ := hashfile.Read(hash)
	if err := bcrypt.CompareHashAndPassword(hash[:n], []byte(args[0])); err != nil {
		session.Conn.Write([]byte("-ERR invalid password\r\n"))
		return
	}
	session.Mailbox.Mu.Lock()
	session.Conn.Write([]byte("+OK maildrop locked and ready\r\n"))
	session.State = TRANS
	session.Pass = args[0]
	session.Conn.SetDeadline(time.Now().Add(5 * time.Minute))
}

// msgs marked as deleted are not included
func commandSTAT(session *ClientSession) {
	if session.State != TRANS && session.State != UPDATE {
		session.Conn.Write([]byte("-ERR action not premitted\r\n"))
		return
	}
	dbPath := path.Join(session.Mailbox.Path, session.Name+".db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		session.Conn.Write([]byte("-ERR database connection failed\r\n"))
		return
	}
	rows, err := db.Query("select length(msg) from mail where is_deleted=0")
	if err != nil {
		session.Conn.Write([]byte("-ERR database query failed\r\n"))
		return
	}
	mailCount := 0
	size := 0
	for rows.Next() {
		l := 0
		rows.Scan(&l)
		mailCount++
		size += l
	}
	session.Conn.Write(fmt.Appendf([]byte{}, "+OK %v %v\r\n", mailCount, size))
}

// msgs marked as deleted are not listed
func commandLIST(session *ClientSession, args []string) {
	if session.State != TRANS && session.State != UPDATE {
		session.Conn.Write([]byte("-ERR action not premitted\r\n"))
		return
	}
	dbPath := path.Join(session.Mailbox.Path, session.Name+".db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		session.Conn.Write([]byte("-ERR database connection failed\r\n"))
		return
	}
	rows, err := db.Query("select id, length(msg), is_deleted from mail")
	if err != nil {
		session.Conn.Write([]byte("-ERR database query failed\r\n"))
		return
	}
	size := 0
	type msg struct {
		id        int
		size      int
		isDeleted int
	}
	msgs := []msg{}
	nonDeletedCount := 0
	for rows.Next() {
		m := msg{}
		rows.Scan(&m.id, &m.size, &m.isDeleted)
		msgs = append(msgs, m)
		size += m.size
		if m.isDeleted == 0 {
			nonDeletedCount++
		}
	}
	if len(args) == 0 {
		session.Conn.Write(fmt.Appendf([]byte{}, "+OK %v messages (%v octets)\r\n", nonDeletedCount, size))
		for i := 0; i < len(msgs); i++ {
			if msgs[i].isDeleted == 0 {
				session.Conn.Write(fmt.Appendf([]byte{}, "%v %v\r\n", msgs[i].id, msgs[i].size))
			}
		}
		session.Conn.Write([]byte(".\r\n"))
	} else {
		id, err := strconv.Atoi(args[0])
		if err != nil {
			session.Conn.Write([]byte("-ERR LIST expects a number\r\n"))
			return
		}
		for i := 0; i < len(msgs); i++ {
			if msgs[i].id == id && msgs[i].isDeleted == 0 {
				session.Conn.Write(fmt.Appendf([]byte{}, "+OK %v %v\r\n", id, msgs[i].size))
				return
			}
		}
		session.Conn.Write([]byte("-ERR no such message\r\n"))
	}
}

func commandRETR(session *ClientSession, args []string) {
	if session.State != TRANS && session.State != UPDATE {
		session.Conn.Write([]byte("-ERR action not premitted\r\n"))
		return
	}
	if len(args) < 1 {
		session.Conn.Write([]byte("-ERR arguments incomplete\r\n"))
		return
	}
	dbPath := path.Join(session.Mailbox.Path, session.Name+".db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		session.Conn.Write([]byte("-ERR database connection failed\r\n"))
		return
	}
	rows, err := db.Query("select msg, length(msg) from mail where is_deleted=0")
	if err != nil {
		session.Conn.Write([]byte("-ERR database query failed\r\n"))
		return
	}
	type msg struct {
		text string
		size int
	}
	msgs := []msg{}
	for rows.Next() {
		text, size := "", 0
		rows.Scan(&text, &size)
		msgs = append(msgs, msg{
			text: text,
			size: size,
		})
	}
	id, err := strconv.Atoi(args[0])
	if err != nil {
		session.Conn.Write([]byte("-ERR RETR expects a number\r\n"))
		return
	}
	id -= 1
	if len(msgs) <= id {
		session.Conn.Write([]byte("-ERR no such message\r\n"))
		return
	}
	session.Conn.Write(fmt.Appendf([]byte{}, "+OK %v octets\r\n", msgs[id].size))
	session.Conn.Write(fmt.Appendf([]byte{}, "%s\r\n", msgs[id].text))
	session.Conn.Write([]byte(".\r\n"))
}

func commandDELE(session *ClientSession, args []string) {
	if session.State != TRANS && session.State != UPDATE {
		session.Conn.Write([]byte("-ERR action not premitted\r\n"))
	}
}

func commandNOOP(session *ClientSession) {
	if session.State != TRANS && session.State != UPDATE {
		session.Conn.Write([]byte("-ERR action not premitted\r\n"))
		return
	}
}

func commandRSET(session *ClientSession) {
	if session.State != TRANS && session.State != UPDATE {
		session.Conn.Write([]byte("-ERR action not premitted\r\n"))
		return
	}
}

func commandTOP(session *ClientSession, args []string) {
	if session.State != TRANS && session.State != UPDATE {
		session.Conn.Write([]byte("-ERR action not premitted\r\n"))
		return
	}
}

func commandUIDL(session *ClientSession, args []string) {
	if session.State != TRANS && session.State != UPDATE {
		session.Conn.Write([]byte("-ERR action not premitted\r\n"))
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
	Generate("test", "test.com", "test")
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
