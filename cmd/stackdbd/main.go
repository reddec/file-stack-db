package main

import (
	"flag"
	"io/ioutil"
	"log"
	"sync"
	"time"

	"github.com/reddec/file-stack-db"
)

var db *fstack.Database

func main() {
	http := flag.String("http", "", "HTTP API endpoint")
	rpc := flag.String("rpc", "", "GO-RPC (gob) endpoint")
	rpcHTTP := flag.String("http-rpc", "", "GO HTTP RPC endpoint. Default prefix will be used")
	rootPath := flag.String("root", "./db", "Root dir for stacked database")
	keepAlive := flag.Duration("keep-alive", 10*time.Second, "Opened file keep-alive timeout")
	silent := flag.Bool("silent", false, "Discard log output")
	flag.Parse()
	if *silent {
		log.SetOutput(ioutil.Discard)
	}
	fsdb, err := fstack.NewDatabase(*rootPath, *keepAlive)
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
	log.Println("Scan done")
	wg := sync.WaitGroup{}
	if *http != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			enableHTTP(*http)
		}()
	}
	if *rpc != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			enableRPC(*rpc)
		}()
	}
	if *rpcHTTP != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			enableRPCHTTP(*rpcHTTP)
		}()
	}
	wg.Wait()
}
