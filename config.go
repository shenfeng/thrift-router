package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	// "rec_router/thrift_rec/rec"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// An AtomicInt is an int64 to be accessed atomically.
type AtomicInt int64

type Backend struct {
	Addr string `json:"addr"`

	Hits      AtomicInt                  `json:"hits"`
	Fails     AtomicInt                  `json:"fails"`     // TCP failed
	Timeouts  AtomicInt                  `json:"timeouts"`  // timeout
	Ongoing   AtomicInt                  `json:"ongoing"`   // on going request
	Dead      AtomicInt                  `json:"dead"`      // is dead
	Latencies [LatencySize]time.Duration `json:"latencies"` //	round buffer

	mu          sync.Mutex //	only used by connections
	connections []net.Conn
}

type TGroup struct {
	Filter     map[string]interface{} `json:"filter"`
	Strategies TStrategies            `json:"strategies"`
	// Filter     map[string]int `json:"filter"`
}

type TStrategy struct {
	Name    string   `json:"name"`
	Traffic float32  `json:"traffic"`
	Copy    string   `json:"copy"`
	Models  []string `json:"models"`
	Timeout int      `json:"timeout"`
	Policy  string   `json:"policy"`

	Hits      AtomicInt                  `json:"hits"`
	Partials  AtomicInt                  `json:"partials"`
	Fails     AtomicInt                  `json:"fails"`
	Timeouts  AtomicInt                  `json:"timeouts"`
	Latencies [LatencySize]time.Duration `json:"latencies"` //	round buffer
}

type TBackendServers []*Backend
type TStrategies []*TStrategy
type TService []*TGroup

type TCConfig struct {
	Servers map[string]TBackendServers
	servers []string `json:"servers"`

	Services map[string]TService `json:"services"`
	loadTime time.Time
}

func (service TService) Choose(t TType, s *Server) *TStrategy {
	// switch v := t.(type) {
	// case *rec.DealRecReq:
	// 	for _, g := range service {
	// 		if g.Filter.matchDealReq(v, s) {
	// 			r := rand.Float32()
	// 			var current float32 = 0
	// 			for _, stra := range g.Strategies {
	// 				if stra.Traffic > 0.001 { //r use set this
	// 					current += stra.Traffic
	// 					if r < current {
	// 						return stra
	// 					}
	// 				} else {
	// 					return stra
	// 				}
	// 			}
	// 		}
	// 	}
	// 	// non match, choose the last one
	// 	ss := service[len(service)-1].Strategies
	// 	return ss[len(ss)-1]
	// case *rec.User_Info: // getRecommendByUser
	// 	return service[0].Strategies[0]
	// case *rec.GetSuggestWithCountArgs: // get_suggest_with_count
	// 	return service[0].Strategies[0]
	// case *rec.GetPersonalGeoArgs:
	// 	for _, g := range service {
	// 		if g.Filter.matchGetPersonalGeoReq(v, s) {
	// 			return g.Strategies[0]
	// 		}
	// 	}
	// 	// can't reach here
	// 	return service[0].Strategies[0]
	// case *rec.SphSearchPoiArgs:
	// 	for _, g := range service {
	// 		if g.Filter.matchPoiSearchReq(v, s) {
	// 			return g.Strategies[0]
	// 		}
	// 	}
	// 	return service[0].Strategies[0]
	// 	// for _, g := range service {
	// 	// 	return g.Strategies[0]
	// 	// }
	// case *rec.GetDealRankArgs:
	// 	for _, g := range service {
	// 		if g.Filter.matchgetDealRankReq(v, s) {
	// 			return g.Strategies[0]
	// 		}
	// 	}
	// 	return service[0].Strategies[0]
	// case *rec.GetRecommendBySearchArgs:
	// 	return service[0].Strategies[0]
	// case *rec.GenSearchExcerptArgs:
	// 	// 搜索摘要服务
	// 	return service[0].Strategies[0]
	// case *rec.SphMultiSearchArgs:
	// 	// 搜索服务
	// 	for _, g := range service {
	// 		if g.Filter.matchSphMultiSearchReq(v, s) {
	// 			return g.Strategies[0]
	// 		}
	// 	}
	// 	return service[0].Strategies[0]
	// case *rec.SphSearchArgs:
	// 	return service[0].Strategies[0]
	// case *rec.SphSearchDealArgs:
	// 	// mobile deal search
	// 	return service[0].Strategies[0]
	// case *rec.GetRecommendByPoiArgs:
	// 	return service[0].Strategies[0]
	// default:
	// 	panic("not understand type")
	// }
}

func readConfigFromBytes(data []byte, dump bool) (*TCConfig, error) {
	cfg := &TCConfig{
		Servers:  make(map[string]TBackendServers),
		Services: make(map[string]TService),
		loadTime: time.Now(),
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	} else {
		for name, servers := range cfg.Servers {
			if len(servers) == 0 {
				return nil, fmt.Errorf("%s has 0 servers", name)
			}
			for _, s := range servers {
				if s.Addr == "" {
					return nil, fmt.Errorf("server addr is empty")
				}
			}
		}
		// check validity
		for _, ss := range cfg.Services {
			for _, g := range ss {
				if g.Filter == nil {
					return nil, fmt.Errorf("filter can not be nil")
				}
				for _, s := range g.Strategies {
					for _, m := range s.Models {
						if cfg.Servers[m] == nil {
							return nil, fmt.Errorf("no server defined for '%v'", m)
						}
					}
					if s.Copy != "" {
						if cfg.Servers[s.Copy] == nil {
							return nil, fmt.Errorf("no server defined for copy: '%v'", s.Copy)
						}
					}
					if s.Name == "" {
						s.Name = strings.Join(s.Models, "-")
					}
				}
			}
		}

		if dump {
			// dump to stdout
			for name, ss := range cfg.Services {
				log.Println("thrift fn name:", name)
				for _, g := range ss {
					log.Println(*g)
				}
			}

			for key, servers := range cfg.Servers {
				log.Println("server group:", key, servers)
			}
		}
		return cfg, nil
	}
}

func readConfig(filename string) (*TCConfig, error) {
	if data, err := ioutil.ReadFile(filename); err == nil {
		return readConfigFromBytes(data, false)
	} else {
		return nil, err
	}
}

// Add atomically adds n to i.
func (i *AtomicInt) Add(n int64) {
	atomic.AddInt64((*int64)(i), n)
}

// Get atomically gets the value of i.
func (i *AtomicInt) Get() int64 {
	return atomic.LoadInt64((*int64)(i))
}

// Get atomically gets the value of i.
func (i *AtomicInt) Set(v int64) {
	atomic.StoreInt64((*int64)(i), v)
}

func (i *AtomicInt) String() string {
	return strconv.FormatInt(i.Get(), 10)
}

// PY_THRIFTH=thrift_rec
// rm -rf $PY_THRIFTH && mkdir $PY_THRIFTH

func (s *Backend) markAsBroken() {
	s.Dead.Set(1)
	s.mu.Lock()
	for _, c := range s.connections {
		c.Close()
	}
	s.connections = nil
	s.mu.Unlock()
}

func (ss TBackendServers) nextOne(prev *Backend) *Backend {
	var min int64 = 100000
	idx := 0
	i := rand.Intn(len(ss))
	limit := i + len(ss)

	for ; i < limit; i++ {
		s := ss[i%len(ss)]
		if s != prev && s.Dead.Get() == 0 && s.Ongoing.Get() < min {
			min = s.Ongoing.Get()
			idx = i % len(ss)
		}
	}
	return ss[idx]
}

func (s *Backend) reAlive() {
	if s.Dead.Get() != 0 {
		if c, err := net.Dial("tcp", s.Addr); err == nil {
			log.Println("server", s.Addr, "is back to service")
			s.Dead.Set(0)
			c.Close()
		}
	}
}

// if return err, should retry sometime later
func (s *Backend) getConn(d time.Duration) (con net.Conn, err error) {
	s.mu.Lock()
	if len(s.connections) > 0 {
		con = s.connections[len(s.connections)-1]
		s.connections = s.connections[0 : len(s.connections)-1]
	}
	s.mu.Unlock()

	if con != nil {
		return
	}

	return net.DialTimeout("tcp", s.Addr, d)
}

func (s *Backend) returnConn(c net.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if c != nil {
		if len(s.connections) >= MaxConPerHost {
			c.Close()
		} else {
			s.connections = append(s.connections, c)
		}
	}
}

func toString(length int, str func(idx int) string) string {
	addrs := make([]string, length)
	for i := 0; i < length; i++ {
		addrs[i] = str(i)
	}
	return fmt.Sprintf("[%s]", strings.Join(addrs, "; "))
}

func (ss TBackendServers) String() string {
	return toString(len(ss), func(i int) string {
		return ss[i].Addr
	})
}

func (ss TStrategies) String() string {
	return toString(len(ss), func(i int) string {
		s := *ss[i]
		return fmt.Sprintf("{models: %s, traffic: %f, timeout: %dms}",
			s.Models, s.Traffic, s.Timeout)
	})
}

func (ss TService) String() string {
	return toString(len(ss), func(i int) string {
		return fmt.Sprintf("{filter: %+v, strategies: %v}",
			ss[i].Filter, ss[i].Strategies)
	})
}

func (t *TCConfig) String() string {
	return fmt.Sprintf("servers:  %v, \nservices: %v", t.Servers, t.Services)
}
