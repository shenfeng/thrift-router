package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"net"
	"thrift"
	"time"
)

type IHooker interface {
	DecodeReq(s *Server, input []byte, seqid int32) ([]byte, TType, error)
	Reduce(input []byte, prev TType, r *Reducer, stra *TStrategy) (bool, TType, error)
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
	request []byte
	reqObj  TType

	result *TResult

	failed     AtomicInt
	conn       net.Conn
	seqId      int32
	fname      string
	stragegy   *TStrategy
	server     *Server
	hooker     IHooker
	start      time.Time
	latency    time.Duration
	serverConf *TCConfig
	done       AtomicInt
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
	if err = writeAll(p.request, client); err != nil {
		return nil, err
	}
	buffer, err = readFramedThrift(client)
	return
}

type TResult struct {
	stragegy *TStrategy
	data     TType
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
			} else if done, r, err := p.hooker.Reduce(msg, partial, p, stragegy); err != nil {
				log.Println("ERROR: reduce", b.Addr, err)
			} else if done {
				if p.done.Get() == 0 {
					stragegy.Latencies[stragegy.Hits.Get()%LatencySize] = time.Since(start)
					ch <- &TResult{stragegy: stragegy, data: r}
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

func (p *Reducer) fetchAndReduce() (*TResult, error) {
	stra := p.stragegy
	stageWait := time.After(time.Millisecond * time.Duration(stra.Timeout))
	// maxWait := time.After(time.Millisecond * time.Duration(stra.Timeout*2))

	if bss, ok := p.serverConf.Servers[stra.Copy]; ok {
		b := bss.nextOne(nil) // 复制线上真实流量到测试机器
		if b.Dead.Get() == 0 {
			go func() {
				p.fetchFromBackend(b)
			}()
		}
	}

	selectedRetry := 1

	// var selectedCh chan *TResult
	selectedCh := p.fetchFromBackends(stra)
	var result *TResult

F:
	for {
		select {
		case <-stageWait:
			// log.Println("timeout")
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
	if result == nil {
		return nil, errors.New("backend server failed")
	}
	result.stragegy.Hits.Add(1)
	return result, nil
}

func formatError(fname string, seqId, exceptionId int32, err error) []byte {
	buffer := thrift.NewTMemoryBuffer()
	trans := thrift.NewTFramedTransport(buffer)
	oprot := thrift.NewTBinaryProtocolTransport(trans)
	a := thrift.NewTApplicationException(exceptionId, err.Error())
	oprot.WriteMessageBegin(fname, thrift.EXCEPTION, seqId)
	a.Write(oprot)
	oprot.WriteMessageEnd()
	oprot.Flush()
	return buffer.Bytes()
}
