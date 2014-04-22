package main

import (
	"compress/gzip"
	"encoding/json"
	_ "expvar"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"sort"
	"strings"
	"time"
)

var indexTmpl *template.Template
var editorTmpl *template.Template

func writeJson(w http.ResponseWriter, data interface{}) {
	json, err := json.Marshal(data)
	if err == nil {
		w.Header().Add("Content-Type", "application/json")
		w.Write([]byte(json))
	} else {
		w.Write([]byte(err.Error()))
	}
}

type ServerStatus struct {
	Server  *Server   `json:"server"`
	Reducer *TCConfig `json:"reducer"`
}

func (p *Server) dumpStatus(w http.ResponseWriter) {
	writeJson(w, &ServerStatus{Server: p, Reducer: p.getConf()})
}

func avgLatency(latencies [LatencySize]time.Duration) time.Duration {
	var d time.Duration = 0
	count := 1
	for _, v := range latencies {
		if v > 0 {
			d += v
			count += 1
		}
	}
	return d / time.Duration(count)
}

func trafficPercent(t float32) string {
	if t > 0 {
		return fmt.Sprintf("%.4f%%", t*100)
	}
	return "Remaining All"
}

func strategyStatus(s *TStrategy, conf *TCConfig) map[string]interface{} {
	policy := s.Policy
	if policy == "" {
		policy = "default"
	}
	copyserver := s.Copy
	if copyserver == "" {
		copyserver = "{nocopy}"
	}
	return map[string]interface{}{
		"name":       s.Name,
		"policy":     policy,
		"timeouts":   s.Timeouts,
		"timeout":    s.Timeout,
		"models":     s.Models,
		"cache":      s.Cache,
		"cachetime":  s.CacheTime,
		"copy":       copyserver,
		"hits":       s.Hits.Get(),
		"partials":   s.Partials.Get(),
		"qps":        fmt.Sprintf("%.2f", float64(s.Hits.Get())/time.Since(conf.loadTime).Seconds()),
		"traffic":    trafficPercent(s.Traffic),
		"fails":      s.Fails.Get(),
		"latencies":  s.Latencies[:ReportLatencySize],
		"avgLatency": avgLatency(s.Latencies),
	}
}

func strategiesStatus(ss TStrategies, conf *TCConfig) []map[string]interface{} {
	rets := make(SortableStatus, len(ss))
	for i, s := range ss {
		rets[i] = strategyStatus(s, conf)
	}
	sort.Sort(rets)
	return rets
}

func servicesStatus(ss map[string]TService, conf *TCConfig) []map[string]interface{} {
	rets := make(SortableStatus, 0, len(ss))
	for name, service := range ss {
		for _, group := range service {
			rets = append(rets, map[string]interface{}{
				"name":       fmt.Sprintf("%s -- %s", name, group.Filter),
				"strategies": strategiesStatus(group.Strategies, conf),
			})
		}
	}
	sort.Sort(rets)

	// rets = append(rets, map[string]interface{}{
	// 	"name":       "fallback"
	// 	"strategies": []map[string]interface{}[strategyStatus()],
	// })

	return rets
}

type SortableStatus []map[string]interface{}

func (sm SortableStatus) Len() int { return len(sm) }
func (sm SortableStatus) Less(i, j int) bool {
	return sm[i]["name"].(string) < sm[j]["name"].(string)
}
func (sm SortableStatus) Swap(i, j int) { sm[i], sm[j] = sm[j], sm[i] }

func backendServerStatus(ss map[string]TBackendServers, conf *TCConfig) []map[string]interface{} {
	rets := make(SortableStatus, 0, len(ss))
	for model, backends := range ss {
		for _, b := range backends {
			rets = append(rets, map[string]interface{}{
				"name":       fmt.Sprintf("name: %s, addr: %s", model, b.Addr),
				"hits":       b.Hits.Get(),
				"fails":      b.Fails.Get(),
				"timeouts":   b.Timeouts.Get(),
				"ongoing":    b.Ongoing.Get(),
				"qps":        float64(b.Hits.Get()) / time.Since(conf.loadTime).Seconds(),
				"dead":       b.Dead.Get() != 0,
				"avgLatency": avgLatency(b.Latencies),
				"latencies":  b.Latencies[:ReportLatencySize],
			})
		}
	}
	sort.Sort(rets)
	return rets
}

func (p *Server) dumpStatusHtml(w http.ResponseWriter) {
	if indexTmpl == nil {
		var err error
		if indexTmpl, err = template.ParseFiles("static/status.tpl"); err != nil {
			fmt.Fprintf(w, "Parse static/status.tpl, error: %s", err)
			return
		}
	}

	now := time.Now()
	f := func(n int64) string {
		return fmt.Sprintf("%d, %.4f%%", n, float64(n)/float64(p.Hits.Get())*100.0)
	}

	conf := p.getConf()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	indexTmpl.Execute(w, map[string]interface{}{
		"start":      p.Start,
		"time":       now.Sub(p.Start),
		"hits":       p.Hits.Get(),
		"qps":        float64(p.Hits.Get()) / time.Since(p.Start).Seconds(),
		"fails":      f(p.Fails.Get()),
		"latencies":  p.Latencies[:ReportLatencySize],
		"avgLatency": avgLatency(p.Latencies),
		"services":   servicesStatus(conf.Services, conf),
		"backends":   backendServerStatus(conf.Servers, conf),
	})
}

func (p *Server) showEditor(w http.ResponseWriter) {
	var err error
	if editorTmpl == nil {
		editorTmpl, err = template.ParseFiles("static/editor.tpl")
	}

	if editorTmpl == nil {
		fmt.Fprintf(w, "Parse static/editor.tpl, error: %s", err)
		return
	}

	data, _ := ioutil.ReadFile(p.conffile)
	editorTmpl.Execute(w, map[string]string{
		"config": string(data),
	})
}

func (p *Server) saveConfig(w http.ResponseWriter, r *http.Request) {
	if e := r.ParseForm(); e == nil {
		config := r.PostFormValue("config")
		if _, err := readConfigFromBytes([]byte(config), false); err == nil {
			log.Println("config file check passed")

			now := time.Now()
			y, m, d := now.Date()
			h, min, s := now.Clock()

			backup := fmt.Sprintf("%s_%d_%d_%d_%d_%d-%d", p.conffile, y, m, d, h, min, s)
			if err := os.Rename(p.conffile, backup); err == nil {
				if e := ioutil.WriteFile(p.conffile, []byte(config), 0666); e == nil {
					log.Println("reload config")
					p.reloadConf()
				} else {
					log.Println("save config failed", e)
				}
			} else {
				log.Println("Rename ERROR", err)
			}
		} else {
			log.Println("config file check failed, ", err)
		}
	} else {
		log.Println("ERROR, parse form", e)
	}
}

func (p *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/":
		p.dumpStatusHtml(w)
	case "/editor":
		switch r.Method {
		case "GET":
			log.Println(r.RemoteAddr, "request show editor")
			p.showEditor(w)
		case "POST":
			log.Println(r.RemoteAddr, "try to save config")
			p.saveConfig(w, r)
		}
	case "/api/status":
		p.dumpStatus(w)
	default:
		http.Redirect(w, r, "/static/", 302)
	}
}

func startHttpAdmin(server *Server, addr string) {
	http.Handle("/", NewHandler(server))
	http.Handle("/static/", NewHandler(http.StripPrefix("/static/", http.FileServer(http.Dir("static")))))

	log.Println("HTTP admin interface:", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
	sniffDone bool
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	if !w.sniffDone {
		if w.Header().Get("Content-Type") == "" {
			w.Header().Set("Content-Type", http.DetectContentType(b))
		}
		w.sniffDone = true
	}
	return w.Writer.Write(b)
}

// Wrap a http.Handler to support transparent gzip encoding.
func NewHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			h.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		defer gz.Close()
		h.ServeHTTP(&gzipResponseWriter{Writer: gz, ResponseWriter: w}, r)
	})
}
