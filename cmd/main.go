package main

import (
	"encoding/json"
	"io/ioutil"
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
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	vars := mux.Vars(r)
	headers := map[string]string{}
	for key, value := range r.Header {
		if strings.HasPrefix("S-", key) {
			headers[key] = value[0]
		}
	}
	stack, err := db.Find(vars["key"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	depth, err := stack.Push(encodeHeaders(headers), data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	sdepth := strconv.Itoa(depth)
	w.Header().Add("Id", sdepth)

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write([]byte(sdepth))
}

func getLast(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	stack, err := db.Find(vars["key"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	headers, body, err := stack.Peak()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	if headers == nil || body == nil {
		http.Error(w, "", http.StatusNotFound)
	}
	sheaders := decodeHeaders(headers)
	for key, value := range sheaders {
		w.Header().Add(key, value)
	}
	w.Header().Add("Count", strconv.Itoa(stack.Depth()))
	w.WriteHeader(200)
	w.Write(body)
}

func removeLast(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	stack, err := db.Find(vars["key"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	headers, body, err := stack.Pop()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	if headers == nil || body == nil {
		http.Error(w, "", http.StatusNotFound)
	}
	sheaders := decodeHeaders(headers)
	for key, value := range sheaders {
		w.Header().Add(key, value)
	}
	w.Header().Add("Count", strconv.Itoa(stack.Depth()))
	w.WriteHeader(200)
	w.Write(body)
}

func main() {
	address := ""
	rootPath := ""
	keepAlive := 10 * time.Second
	fsdb, err := fstack.NewDatabase(rootPath, keepAlive)
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
	router.Methods("GET").Path("/:key").HandlerFunc(getLast)
	router.Methods("POST").Path("/:key").HandlerFunc(pushData)
	router.Methods("DELete").Path("/:key").HandlerFunc(removeLast)
	panic(http.ListenAndServe(address, router))
}
