package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"net"
	// "thrift"
	"time"
)

type IHooker interface {
	DecodeReq(s *Server, input []byte, seqid int32) (*TRequest, error)
	Reduce(input []byte, prev TType, r *Reducer) (bool, TType, error)
	// Encode(result TType, r *Reducer) []byte
	Log(r *Reducer)
}

func writeAll(message []byte, c net.Conn) error {
	start, end := 0, len(message)
	for start < end {
		n, err := c.Write(message[start:])
		if err != nil {
			return err
		} else {
			start += n
		}
	}
	return nil
}

func readFramedThrift(c net.Conn) ([]byte, error) {
	lenBuf := [4]byte{}

	// buffer := make([]byte, BufferSize)
	read := 0
	for read < 4 {
		n, err := c.Read(lenBuf[read:])
		if err != nil {
			return nil, err
		}
		if n == 0 {
			log.Println("read error", n)
		}

		read += n
	}
	length := int(binary.BigEndian.Uint32(lenBuf[:]))
	if length > MaxRequestSize {
		return nil, fmt.Errorf("message is too big, %d", length)
	}

	// TODO is size be a fixes better?a
	buffer := make([]byte, length+4)
	copy(buffer, lenBuf[:])

	for read < length+4 {
		n, err := c.Read(buffer[read:])
		if err != nil {
			return nil, err
		}
		read += n
	}
	return buffer, nil
}

type Reducer struct {
	req    *TRequest
	result *TResult

	failed     AtomicInt
	done       AtomicInt
	seqId      int32
	fname      string
	conn       net.Conn
	stragegy   *TStrategy
	server     *Server
	hooker     IHooker
	start      time.Time
	latency    time.Duration
	serverConf *TCConfig
}

func (p *Reducer) fetchFromBackend(b *Backend) (buffer []byte, err error) {
	var client net.Conn

	b.Ongoing.Add(1)
	b.Hits.Add(1)
	start := time.Now()

	defer func() {
		if err != nil {
			log.Println("server broken", b.Addr, err)
			b.markAsBroken()
			// p.err = err
			// log.Println(b.addr, err)
			b.Fails.Add(1)
			if client != nil {
				client.Close()
			}
		} else {
			b.returnConn(client)
		}
		b.Ongoing.Add(-1)
		latency := time.Since(start)
		b.addLatency(latency)
	}()
	if client, err = b.getConn(time.Millisecond * 100); err != nil {
		return nil, err
	}

	client.SetDeadline(start.Add(time.Minute * 1))
	if err = writeAll(p.req.bytes, client); err != nil {
		return nil, err
	}
	buffer, err = readFramedThrift(client)
	return
}

type TResult struct {
	stragegy *TStrategy
	data     TType
	bytes    []byte
}

type TRequest struct {
	bytes    []byte
	obj      TType
	cacheKey string
}

func (p *Reducer) fetchFromBackends(stragegy *TStrategy) chan *TResult {
	ch := make(chan *TResult, 1)
	go func() {
		var partial TType
		for _, server := range stragegy.Models {
			b := p.serverConf.Servers[server].nextOne(nil)
			if b.Dead.Get() != 0 { // ignore dead server
				continue
			}
			start := time.Now()
			if msg, err := p.fetchFromBackend(b); err != nil {
				log.Println("ERROR: fetch", b.Addr, err)
			} else if done, r, err := p.hooker.Reduce(msg, partial, p); err != nil {
				log.Println("ERROR: reduce", b.Addr, err)
			} else if done {
				if p.done.Get() == 0 {
					stragegy.Latencies[stragegy.Hits.Get()%LatencySize] = time.Since(start)
					ch <- &TResult{stragegy: stragegy, data: r, bytes: binaryProtocolEncode(p, r)}
				} else {
					stragegy.Timeouts.Add(1)
				}
				return
			} else {
				partial = r
			}
		}
		if partial == nil {
			stragegy.Fails.Add(1)
			ch <- nil
		} else {
			stragegy.Partials.Add(1)
			ch <- &TResult{stragegy: stragegy, data: partial}
		}
	}()
	return ch
}

func (p *Reducer) fetchAndReduce() error {
	stra := p.stragegy

	if bss, ok := p.serverConf.Servers[stra.Copy]; ok {
		b := bss.nextOne(nil) // 复制线上真实流量到测试机器
		if b.Dead.Get() == 0 {
			go func() {
				p.fetchFromBackend(b)
			}()
		}
	}

	stra.Hits.Add(1)
	if stra.Cache != nil && stra.CacheTime > 0 {
		if r, ok := stra.Cache.Get(p.req.cacheKey); ok {
			p.result = &TResult{stragegy: stra, bytes: copyAndfixSeqId(r, p.seqId)}
			return nil
		}
	}

	timeout := time.After(time.Millisecond * time.Duration(stra.Timeout))
	selectedRetry := 1

	// var selectedCh chan *TResult
	selectedCh := p.fetchFromBackends(stra)
	var result *TResult

F:
	for {
		select {
		case <-timeout:
			break F // fail
		case result = <-selectedCh:
			if result == nil {
				selectedRetry += 1
				if selectedRetry <= MaxRetry {
					selectedCh = p.fetchFromBackends(stra) // retry
				} else {
					break F // fail
				}
			} else {
				break F // done
			}
		}
	}

	p.done.Set(1)
	p.result = result
	if result == nil {
		return errors.New("backend server failed")
	}

	if stra.Cache != nil && stra.CacheTime > 0 {
		stra.Cache.Setex(p.req.cacheKey, stra.CacheTime, result.bytes)
	}

	return nil
}
