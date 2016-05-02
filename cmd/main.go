package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/reddec/file-stack-db"
)

var db *fstack.Database

func encodeHeaders(headers map[string]string) []byte {
	v, err := json.Marshal(headers)
	if err != nil {
		panic(err)
	}
	return v
}

func decodeHeaders(data []byte) map[string]string {
	var v map[string]string
	err := json.Unmarshal(data, &v)
	if err != nil {
		panic(err)
	}
	return v
}

func pushData(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println("[PUSH]", "Failed read body from request for stack", vars["key"], err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	headers := map[string]string{}
	for key, value := range r.Header {
		if strings.HasPrefix("S-", key) {
			headers[key] = value[0]
		}
	}
	stack, err := db.Find(vars["key"], true)
	if err != nil {
		log.Println("[PUSH]", "Failed find stack", vars["key"], err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	binHeaders := encodeHeaders(headers)

	depth, err := stack.Push(binHeaders, data)
	if err != nil {
		log.Println("[PUSH]", "Failed push to", vars["key"], err)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	sdepth := strconv.Itoa(depth)
	log.Println("[PUSH]", "Pushed", len(data), "bytes with headers", len(binHeaders), "bytes to", vars["key"], "with depth-index", sdepth)
	w.Header().Add("Id", sdepth)

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write([]byte(sdepth))
}

func getLast(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	stack, err := db.Find(vars["key"], false)
	if err != nil {
		log.Println("[PEAK]", "Failed find stack", vars["key"], err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if stack == nil {
		log.Println("[PEAK]", "Stack", vars["key"], "not exists")
		http.Error(w, "", http.StatusNotFound)
		return
	}
	headers, body, err := stack.Peak()
	if err != nil {
		log.Println("[PEAK]", "Failed peak stack", vars["key"], err)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	if headers == nil || body == nil {
		log.Println("[PEAK]", "Stack", vars["key"], "is empty")
		http.Error(w, "", http.StatusNotFound)
		return
	}
	sheaders := decodeHeaders(headers)
	for key, value := range sheaders {
		w.Header().Add(key, value)
	}
	log.Println("[PEAK]", "Read stack", vars["key"], "headers:", len(headers), "bytes, body:", len(body), "bytes")
	w.Header().Add("Count", strconv.Itoa(stack.Depth()))
	w.WriteHeader(200)
	w.Write(body)
}

func removeLast(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	stack, err := db.Find(vars["key"], false)
	if err != nil {
		log.Println("[POP]", "Failed find stack", vars["key"], err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if stack == nil {
		log.Println("[POP]", "Stack", vars["key"], "not exists")
		http.Error(w, "", http.StatusNotFound)
		return
	}
	headers, body, err := stack.Pop()
	if err != nil {
		log.Println("[POP]", "Failed pop stack", vars["key"], err)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	if headers == nil || body == nil {
		log.Println("[POP]", "Stack", vars["key"], "is empty")
		http.Error(w, "", http.StatusNotFound)
		return
	}
	sheaders := decodeHeaders(headers)
	for key, value := range sheaders {
		w.Header().Add(key, value)
	}
	log.Println("[POP]", "Read stack", vars["key"], "headers:", len(headers), "bytes, body:", len(body), "bytes")
	w.Header().Add("Count", strconv.Itoa(stack.Depth()))
	w.WriteHeader(200)
	w.Write(body)
}

func main() {
	address := flag.String("http", ":9000", "Basic HTTP API endpoint")
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
	err = db.Scan()
	if err != nil {
		panic(err)
	}
	router := mux.NewRouter()
	router.Methods("GET").Path("/{key}").HandlerFunc(getLast)
	router.Methods("POST").Path("/{key}").HandlerFunc(pushData)
	router.Methods("DELete").Path("/{key}").HandlerFunc(removeLast)
	http.Handle("/", router)
	panic(http.ListenAndServe(*address, nil))
}
