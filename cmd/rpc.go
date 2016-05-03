package main

import (
	"log"
	"net"
	"net/http"
	"net/rpc"
	"strings"

	"github.com/reddec/file-stack-db/api"
)

type Service struct {
	api.Service
}

func (srv *Service) Push(msg api.PushArgs, resultDepthIndex *int) error {
	log.Println("[RPC] Push to", msg.Section, "headers:", len(msg.Headers), "items, body:", len(msg.Body), "bytes")
	s, err := db.Find(msg.Section, true)
	if err != nil {
		return err
	}
	binHeaders := encodeHeaders(msg.Headers)
	id, err := s.Push(binHeaders, msg.Body)
	if err != nil {
		return err
	}
	*resultDepthIndex = id
	return nil
}

func (srv *Service) Peak(section string, result *api.DataResult) error {
	log.Println("[RPC] Peak from", section)
	s, err := db.Find(section, false)
	if err != nil {
		return err
	}
	if s == nil {
		return api.ErrSectionNotFound
	}
	dr := api.DataResult{}
	dr.DepthIndex = s.Depth()
	headers, body, err := s.Peak()
	if err != nil {
		return err
	}
	if headers == nil || body == nil {
		return api.ErrStackIsEmpty
	}
	dr.Headers = decodeHeaders(headers)
	dr.Body = body
	*result = dr
	return nil
}

func (srv *Service) Pop(section string, result *api.DataResult) error {
	log.Println("[RPC] Pop from", section)
	s, err := db.Find(section, false)
	if err != nil {
		return err
	}
	if s == nil {
		return api.ErrSectionNotFound
	}
	dr := api.DataResult{}
	dr.DepthIndex = s.Depth()
	headers, body, err := s.Pop()
	if err != nil {
		return err
	}
	if headers == nil || body == nil {
		return api.ErrStackIsEmpty
	}
	dr.Headers = decodeHeaders(headers)
	dr.Body = body
	*result = dr
	return nil
}

func (srv *Service) Sections(prefix string, result *[]api.Section) error {
	log.Println("[RPC] Sections with prefix", prefix)
	res := []api.Section{}
	names := db.Names()
	for _, name := range names {
		if strings.HasPrefix(name, prefix) {
			var sec api.Section
			s := db.Get(name)
			sec.Depth = s.Depth()
			sec.LastAccess = s.LastAccess()
			sec.Name = name
			res = append(res, sec)
		}
	}
	*result = res
	return nil
}

func enableRPC(endpoint string) {
	srv := new(Service)
	rpc.RegisterName("db", srv)

	l, e := net.Listen("tcp", endpoint)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	rpc.Accept(l)
}

func enableRPCHTTP(endpoint string) {
	srv := new(Service)
	server := rpc.NewServer()
	server.RegisterName("db", srv)
	l, e := net.Listen("tcp", endpoint)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	panic(http.Serve(l, server))
}
