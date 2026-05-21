package main

import (
	"net"
	"testing"
	"time"
)

func TestMain(t *testing.M) {
	go func() {
		StartServer()
	}()
	time.Sleep(500 * time.Millisecond)
	t.Run()
}

func TestPop3Server(t *testing.T) {
	// tests for auth
	testCases := []struct {
		name string
		in   string
		out  string
	}{
		{
			name: "USER",
			in:   "user test@test.com\r\n",
			out:  "+OK test@test.com is a valid mailbox\r\n",
		},
		{
			name: "PASS",
			in:   "pass test\r\n",
			out:  "+OK maildrop locked and ready\r\n",
		},
	}
	conn, err := net.DialTimeout("tcp", "localhost:5000", 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	data := make([]byte, 1024)
	n, err := conn.Read(data)
	if err != nil {
		t.Fatal(err)
	}
	expected := "+OK POP3 server ready\r\n"
	if string(data[:n]) != expected {
		t.Errorf("GREET: expected %v, got: %v", expected, string(data[:n]))
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			conn.Write([]byte(tc.in))
			data := make([]byte, 1024)
			n, err := conn.Read(data)
			if err != nil {
				t.Fatal(err)
			}
			if string(data[:n]) != tc.out {
				t.Errorf("%v: expected: %v, got: %v", tc.name, tc.out, string(data[:n]))
			}
		})
	}
	// tests for trans
	testCases = []struct {
		name string
		in   string
		out  string
	}{
		{
			name: "STAT",
			in:   "stat\r\n",
			out:  "+OK 1 9\r\n",
		},
		{
			name: "LIST",
			in:   "list\r\n",
			out:  "+OK 1 messages (9 octets)\r\n1 9\r\n.\r\n",
		},
		{
			name: "LIST",
			in:   "list 1\r\n",
			out:  "+OK 1 9\r\n",
		},
		{
			name: "RETR",
			in:   "retr 1\r\n",
			out:  "+OK 9 octets\r\ntest mail\r\n.\r\n",
		},
		{
			name: "DELE",
			in:   "dele 1\r\n",
			out:  "+OK message 1 deleted\r\n",
		},
		{
			name: "NOOP",
			in:   "noop\r\n",
			out:  "+OK\r\n",
		},
		{
			name: "RSET",
			in:   "rset\r\n",
			out:  "+OK\r\n",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			conn.Write([]byte(tc.in))
			data := make([]byte, 1024)
			n, err := conn.Read(data)
			if err != nil {
				t.Fatal(err)
			}
			if string(data[:n]) != tc.out {
				t.Errorf("%v: expected: %v, got: %v", tc.name, tc.out, string(data[:n]))
			}
		})
	}
}
