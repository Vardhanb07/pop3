package main

import (
	"log"
	"os"
	"path"

	"golang.org/x/crypto/bcrypt"
)

func Generate(name, domain, pass string) {
	hdir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	mailbox := path.Join(hdir, ".pop3", domain, name)
	if _, err := os.Open(mailbox); os.IsNotExist(err) {
		if _, err := os.Create(mailbox); err != nil {
			log.Fatal(err)
		}
	}
	hashfilepath := path.Join(hdir, ".pop3", "."+name+"_hash")
	if _, err := os.Open(hashfilepath); os.IsNotExist(err) {
		if _, err := os.Create(hashfilepath); err != nil {
			log.Fatal(err)
		}
	}
	hashfile, err := os.Open(hashfilepath)
	if err != nil {
		log.Fatal(err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal(err)
	}
	hashfile.Write(hash)
}
