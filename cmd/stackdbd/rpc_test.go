package main

import (
	"log"
	"net/rpc"
	"testing"
	"time"

	"github.com/reddec/file-stack-db"
	"github.com/reddec/file-stack-db/api"
)

func TestRPCClient(t *testing.T) {
	fsdb, err := fstack.NewDatabase("test-data/db", 3*time.Second)
	if err != nil {
		panic(err)
	}
	db = fsdb
	defer db.Close()
	log.Println("Scaning saved stacks")
	err = db.Scan()
	if err != nil {
		panic(err)
	}

	go enableRPC(":29900")
	time.Sleep(1 * time.Second)
	client, err := rpc.Dial("tcp", "127.0.0.1:29900")
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	push := api.PushArgs{}
	push.Section = "test"
	push.Headers = map[string]string{"Name": "Alex"}
	push.Body = []byte("Hello world")
	var depth int
	err = client.Call("db.Push", push, &depth)
	if err != nil {
		t.Fatal(err)
	}

	var names []api.Section

	err = client.Call("db.Sections", "te", &names)
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 1 {
		t.Fatal("Invalid count of sections")
	}
	if names[0].Name != "test" {
		t.Fatal("Bad section name")
	}

	var data api.DataResult

	err = client.Call("db.Peak", "test", &data)
	if err != nil {
		t.Fatal(err)
	}

	if data.DepthIndex != depth {
		log.Fatal("Bad depth index on peak")
	}

	err = client.Call("db.Pop", "test", &data)
	if err != nil {
		t.Fatal(err)
	}

	if data.DepthIndex != depth {
		log.Fatal("Bad depth index on pop")
	}

}
