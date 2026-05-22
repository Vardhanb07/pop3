package main

import (
	"database/sql"
	"log"
	"os"
	"path"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

// this functions creates the files .test_hash, test.db, test.lock in the dir ~/.pop3/test.com/ for the mail test@test.com
func Generate(name, domain, pass string) {
	hdir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	mailbox := path.Join(hdir, ".pop3", domain, name+".db")
	if _, err := os.Open(mailbox); os.IsNotExist(err) {
		if err := os.Mkdir(path.Join(hdir, ".pop3", domain), 0777); os.IsNotExist(err) {
			log.Fatal(err)
		}
		if _, err := os.Create(mailbox); err != nil {
			log.Fatal(err)
		}
	}
	hashfilepath := path.Join(hdir, ".pop3", domain, "."+name+"_hash")
	if _, err := os.Open(hashfilepath); os.IsNotExist(err) {
		if _, err := os.Create(hashfilepath); err != nil {
			log.Fatal(err)
		}
	}
	hashfile, err := os.OpenFile(hashfilepath, os.O_RDWR, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer hashfile.Close()
	lockfilepath := path.Join(hdir, ".pop3", domain, name+".lock")
	if _, err := os.Open(lockfilepath); os.IsNotExist(err) {
		if _, err := os.Create(lockfilepath); err != nil {
			log.Fatal(err)
		}
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := hashfile.Write(hash); err != nil {
		log.Fatal(err)
	}
	db, err := sql.Open("sqlite", mailbox)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec("drop table if exists mail"); err != nil {
		log.Fatal(err)
	}
	if _, err := db.Exec("create table mail (id integer primary key autoincrement, msg text not null, is_deleted boolean default false)"); err != nil {
		log.Fatal(err)
	}
	if _, err := db.Exec("insert into mail ('msg', 'is_deleted') values ('test mail', 0)"); err != nil {
		log.Fatal(err)
	}
}
