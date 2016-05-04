package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/rpc"
	"os"
	"strings"
	"time"

	"github.com/reddec/file-stack-db/api"
)

func main() {
	if len(os.Args) < 3 {
		usage()
	}
	addr := os.Args[2]
	http := false
	var err error
	var client *rpc.Client
	if http {
		client, err = rpc.DialHTTP("tcp", addr)
	} else {
		client, err = rpc.Dial("tcp", addr)
	}
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()
	switch os.Args[1] {
	case "push":
		push(client)
	case "pop":
		pop(client)
	case "peak":
		peak(client)
	case "sections":
		sections(client)
	default:
		usage()
	}
}

func usage() {
	fmt.Println(`
Command line access to file stack database
Commands:

  push     <address> <section> [headers=value ...] - push data to file-stack-db
  peak     <address> <section>                     - get last data
  pop      <address> <section>                     - get and remove last data
  sections <address> <prefix >                     - get section info filtered by prefix`)
	os.Exit(1)
}

func push(client *rpc.Client) {
	var args api.PushArgs
	args.Section = os.Args[3]
	args.Headers = make(map[string]string)
	for _, arg := range os.Args[4:] {
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) != 2 {
			continue
		}
		args.Headers[parts[0]] = parts[1]
	}
	data, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		log.Fatal(err)
	}
	args.Body = data
	var id int
	err = client.Call("db.Push", args, &id)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(id)
}

func pop(client *rpc.Client) {
	var data api.DataResult
	err := client.Call("db.Pop", os.Args[3], &data)
	if err != nil {
		log.Fatal(err)
	}
	printSingleMessage(data)
}

func peak(client *rpc.Client) {
	var data api.DataResult
	err := client.Call("db.Peak", os.Args[3], &data)
	if err != nil {
		log.Fatal(err)
	}
	printSingleMessage(data)
}

func printSingleMessage(data api.DataResult) {
	for k, v := range data.Headers {
		fmt.Fprintf(os.Stderr, "%s=%s\n", k, v)
	}
	os.Stdout.Write(data.Body)
}

func sections(client *rpc.Client) {
	var secs []api.Section
	var prefix string
	if len(os.Args) < 4 {
		prefix = ""
	} else {
		prefix = os.Args[3]
	}
	err := client.Call("db.Sections", prefix, &secs)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Fprintln(os.Stderr, "num", "name", "depth", "last-access")
	for id, sec := range secs {
		fmt.Println(id, sec.Name, sec.Depth, sec.LastAccess.Format(time.RFC3339Nano))
	}
}
