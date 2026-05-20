package main

import (
	"os"
	"path"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestGenerate(t *testing.T) {
	name, domain, password := "name", "domain", "password"
	Generate(name, domain, password)
	hdir, _ := os.UserHomeDir()
	mailbox := path.Join(hdir, ".pop3", domain, name+".db")
	if _, err := os.Open(mailbox); os.IsNotExist(err) {
		t.Errorf("Greeting(%v, %v, %v) should create mailbox %v", name, domain, password, mailbox)
	}
	hashfilepath := path.Join(hdir, ".pop3", domain, "."+name+"_hash")
	if _, err := os.Open(hashfilepath); err != nil {
		t.Errorf("Greeting(%v, %v, %v) should create .%v_hash", name, domain, password, name)
	}
	hashfile, _ := os.Open(hashfilepath)
	hash := make([]byte, 1024)
	n, _ := hashfile.Read(hash)
	if bcrypt.CompareHashAndPassword(hash[:n], []byte(password)) != nil {
		t.Errorf("Greeting(%v, %v, %v) should create .%v_hash file which contains a hash", name, domain, password, name)
	}
}
