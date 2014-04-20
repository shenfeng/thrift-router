package main

import (
	"encoding/binary"
	"errors"
	"flag"
	// "fmt"
	"io"
	// "io/ioutil"
	"log"
	"net"
	// "os"
	"runtime"
	// "strconv"
	// "strings"
	"sync/atomic"
	"thrift"
	"time"
	"unsafe"
)

const (
	LatencySize       = 128
	MaxConPerHost     = 5
	MaxRequestSize    = 1024 * 128 // 128k
	MaxRetry          = 2
	ReportLatencySize = 14
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func (s *Backend) addLatency(d time.Duration) {
	s.Latencies[s.Hits.Get()%LatencySize] = d
}

type Server struct {
	conf unsafe.Pointer

	Hits      AtomicInt                  `json:"hits"`
	Fails     AtomicInt                  `json:"fails"`
	Latencies [LatencySize]time.Duration `json:"latencies"`

	Start    time.Time `json:"start"`
	conffile string
}

func (s *Server) reloadConf() error {
	if cfg, err := readConfig(s.conffile); err != nil {
		return err
	} else {
		atomic.StorePointer(&s.conf, unsafe.Pointer(cfg))
		log.Printf("reload config file ok")
		return nil
	}
}

// common interface for thrift genereated struct
type TType interface {
	Read(prot thrift.TProtocol) error
	Write(oprot thrift.TProtocol) error
}

func (s *Server) getConf() *TCConfig {
	return (*TCConfig)(atomic.LoadPointer(&s.conf))
}

func (s *Server) doReduce(request []byte, start time.Time) (r *Reducer, errr error) {
	iprot := thrift.NewTRBinaryProtocol(request[4:])
	name, _, seqId, err := iprot.ReadMessageBegin()
	if err != nil {
		return nil, err
	}

	conf := s.getConf()
	if groups, ok := conf.Services[name]; ok {
		hooker := NewHooker(name)
		if request, obj, err := hooker.DecodeReq(s, request, seqId); err != nil {
			return nil, err
		} else {
			r = &Reducer{
				hooker:     hooker,
				request:    request,
				start:      start,
				reqObj:     obj,
				seqId:      seqId,
				fname:      name,
				server:     s,
				serverConf: conf,
				stragegy:   groups.Choose(obj, s),
			}
			if result, err := r.fetchAndReduce(); err == nil {
				r.result = result
				return r, nil
			} else {
				return nil, err
			}
		}
	} else {
		return nil, errors.New("Unknow function " + name)
	}
}

func (s *Server) checkDeadServers() {
	c := time.Tick(400 * time.Millisecond)
	for _ = range c {
		conf := s.getConf()
		for _, ss := range conf.Servers {
			for _, s := range ss {
				s.reAlive()
			}
		}
	}
}

func encode(r *Reducer) []byte {
	t := r.result.data

	oprot := thrift.NewTWBinaryProtocol(16 * 1024)
	oprot.WriteMessageBegin(r.fname, thrift.REPLY, r.seqId)
	t.Write(oprot)
	oprot.WriteMessageEnd()
	data := oprot.Bytes()

	buffer := make([]byte, len(data)+4)
	binary.BigEndian.PutUint32(buffer, uint32(len(data)))
	copy(buffer[4:], data)
	return buffer
}

func (s *Server) handle(c net.Conn) {
	defer c.Close()
	for {
		start := time.Now()
		message, err := readFramedThrift(c)
		if err != nil {
			if err != io.EOF {
				log.Println("reading thrift:", err)
			}
			return
		}
		s.Hits.Add(1)
		r, err := s.doReduce(message, start)
		if err == nil {
			buffer := encode(r)
			writeAll(buffer, c)
			r.latency = time.Since(start)
			s.Latencies[s.Hits.Get()%LatencySize] = r.latency
			// if s.logfile != nil {
			// 	r.hooker.Log(r, s.logfile)
			// }
		} else {
			buffer := formatError(r.fname, r.seqId, thrift.UNKNOWN_APPLICATION_EXCEPTION, err)
			writeAll(buffer, c)
			s.Fails.Add(1)
			return
		}
	}
}

func main() {
	var conf, addr, httpAdmin string
	var test bool
	flag.StringVar(&addr, "addr", "0.0.0.0:6666", "Which Addr the proxy listens")
	flag.StringVar(&httpAdmin, "http", "0.0.0.0:6060", "HTTP admin addr")
	flag.StringVar(&conf, "conf", "config.json", "Config file path")
	flag.BoolVar(&test, "test", false, "Test config file and exits")
	flag.Parse()

	server := &Server{Start: time.Now(), conffile: conf}
	if err := server.reloadConf(); err != nil {
		log.Fatal(err)
	}

	if test {
		log.Println("config file test pass")
		return
	}

	ln, err := net.Listen("tcp", addr)
	log.Println("thrift listen on", addr)
	if err != nil {
		log.Fatal(err)
	}

	go server.checkDeadServers()

	// go startHttpAdmin(server, httpAdmin)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		go server.handle(conn)
	}
}
