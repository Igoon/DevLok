package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net"
	"net/http"
	"path"
	"reflect"
	"sort"
	"strings"
	"sync"
)

// Options mengonfigurasi server API browser.
type Options struct {
	// Target adalah pointer ke struct yang method exported-nya diekspos
	// sebagai endpoint JSON. Biasanya instance App GUI.
	Target any
	// Assets adalah root aset frontend (berisi index.html dan folder assets).
	Assets fs.FS
	// Addr adalah alamat listen, contoh "127.0.0.1:0" (port otomatis).
	Addr string
}

// Server menyajikan frontend dan menjembatani binding Go ke HTTP.
type Server struct {
	target  reflect.Value
	methods map[string]reflect.Method
	assets  fs.FS

	mu      sync.Mutex
	clients map[chan event]struct{}
}

type event struct {
	Event string `json:"event"`
	Data  any    `json:"data"`
}

type callRequest struct {
	Method string            `json:"method"`
	Args   []json.RawMessage `json:"args"`
}

type callResponse struct {
	Result any    `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

// New membuat server untuk target dan aset yang diberikan.
func New(opts Options) (*Server, error) {
	if opts.Target == nil {
		return nil, errors.New("api: Target wajib diisi")
	}
	tv := reflect.ValueOf(opts.Target)
	if tv.Kind() != reflect.Ptr || tv.IsNil() {
		return nil, errors.New("api: Target harus pointer non-nil")
	}
	t := tv.Type()
	methods := map[string]reflect.Method{}
	for i := 0; i < t.NumMethod(); i++ {
		m := t.Method(i)
		if m.PkgPath != "" {
			continue
		}
		methods[m.Name] = m
	}
	return &Server{
		target:  tv,
		methods: methods,
		assets:  opts.Assets,
		clients: map[chan event]struct{}{},
	}, nil
}

// Handler mengembalikan http.Handler untuk diuji atau dipasang.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/__devlok_shim.js", s.handleShim)
	mux.HandleFunc("/api/call", s.handleCall)
	mux.HandleFunc("/api/events", s.handleEvents)
	mux.HandleFunc("/", s.handleStatic)
	return mux
}

// Listen membuka listener TCP pada alamat yang diminta.
func (s *Server) Listen(addr string) (net.Listener, error) {
	if addr == "" {
		addr = "127.0.0.1:0"
	}
	return net.Listen("tcp", addr)
}

// Serve melayani request sampai listener ditutup.
func (s *Server) Serve(ln net.Listener) error {
	srv := &http.Server{Handler: s.Handler()}
	return srv.Serve(ln)
}

// Emit mengirim event ke semua klien SSE yang terhubung.
func (s *Server) Emit(name string, data any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for ch := range s.clients {
		select {
		case ch <- event{Event: name, Data: data}:
		default:
		}
	}
}

// Methods mengembalikan daftar nama method yang diekspos (terurut).
func (s *Server) Methods() []string {
	out := make([]string, 0, len(s.methods))
	for name := range s.methods {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func (s *Server) handleCall(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, callResponse{Error: "Metode HTTP tidak didukung."})
		return
	}
	var req callRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 8<<20)).Decode(&req); err != nil {
		writeJSON(w, callResponse{Error: "Permintaan tidak valid."})
		return
	}
	m, ok := s.methods[req.Method]
	if !ok {
		writeJSON(w, callResponse{Error: "Metode tidak dikenal: " + req.Method})
		return
	}
	if m.Type.IsVariadic() {
		writeJSON(w, callResponse{Error: "Metode variadic tidak didukung: " + req.Method})
		return
	}
	numIn := m.Type.NumIn() - 1
	if len(req.Args) != numIn {
		writeJSON(w, callResponse{Error: fmt.Sprintf("%s: butuh %d argumen, dapat %d", req.Method, numIn, len(req.Args))})
		return
	}
	in := make([]reflect.Value, numIn)
	for i := 0; i < numIn; i++ {
		ptr := reflect.New(m.Type.In(i + 1))
		if err := json.Unmarshal(req.Args[i], ptr.Interface()); err != nil {
			writeJSON(w, callResponse{Error: fmt.Sprintf("Argumen %d tidak valid: %v", i+1, err)})
			return
		}
		in[i] = ptr.Elem()
	}
	out, callErr := s.invoke(m, in)
	if callErr != nil {
		writeJSON(w, callResponse{Error: callErr.Error()})
		return
	}
	result, err := splitResult(out)
	if err != nil {
		writeJSON(w, callResponse{Error: err.Error()})
		return
	}
	writeJSON(w, callResponse{Result: result})
}

func (s *Server) invoke(m reflect.Method, in []reflect.Value) (out []reflect.Value, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	args := make([]reflect.Value, 0, len(in)+1)
	args = append(args, s.target)
	args = append(args, in...)
	out = m.Func.Call(args)
	return out, nil
}

var errorType = reflect.TypeOf((*error)(nil)).Elem()

func splitResult(out []reflect.Value) (any, error) {
	if len(out) == 0 {
		return nil, nil
	}
	last := out[len(out)-1]
	if last.Type().Implements(errorType) {
		if !last.IsNil() {
			return nil, last.Interface().(error)
		}
		out = out[:len(out)-1]
	}
	switch len(out) {
	case 0:
		return nil, nil
	case 1:
		return out[0].Interface(), nil
	default:
		vals := make([]any, len(out))
		for i, v := range out {
			vals[i] = v.Interface()
		}
		return vals, nil
	}
}

func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	if s.assets == nil {
		http.Error(w, "aset frontend tidak tersedia", http.StatusInternalServerError)
		return
	}
	p := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
	if p == "" || p == "." {
		p = "index.html"
	}
	content, err := fs.ReadFile(s.assets, p)
	if err != nil {
		content, err = fs.ReadFile(s.assets, "index.html")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		p = "index.html"
	}
	if p == "index.html" {
		content = injectShim(content)
	}
	if ct := mime.TypeByExtension(path.Ext(p)); ct != "" {
		w.Header().Set("Content-Type", ct)
	}
	_, _ = w.Write(content)
}

func (s *Server) handleShim(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	names, _ := json.Marshal(s.Methods())
	_, _ = w.Write([]byte(strings.Replace(shimTemplate, "__METHODS__", string(names), 1)))
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	fl, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming tidak didukung", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	ch := make(chan event, 32)
	s.mu.Lock()
	s.clients[ch] = struct{}{}
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		delete(s.clients, ch)
		s.mu.Unlock()
	}()
	_, _ = io.WriteString(w, ": connected\n\n")
	fl.Flush()
	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case ev := <-ch:
			b, err := json.Marshal(ev)
			if err != nil {
				continue
			}
			_, _ = fmt.Fprintf(w, "data: %s\n\n", b)
			fl.Flush()
		}
	}
}

func writeJSON(w http.ResponseWriter, v callResponse) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(v)
}

func injectShim(index []byte) []byte {
	shim := []byte(`<script src="/__devlok_shim.js"></script>`)
	html := string(index)
	if i := strings.Index(html, "</head>"); i >= 0 {
		return []byte(html[:i] + string(shim) + html[i:])
	}
	if i := strings.Index(html, "</body>"); i >= 0 {
		return []byte(html[:i] + string(shim) + html[i:])
	}
	return append(index, shim...)
}

const shimTemplate = `(function () {
  var methods = __METHODS__;
  function rpc(method, args) {
    return fetch('/api/call', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ method: method, args: args })
    }).then(function (res) {
      return res.json().catch(function () { return { error: 'Respons server tidak valid.' }; });
    }).then(function (data) {
      if (data && data.error) { throw new Error(data.error); }
      return data ? data.result : undefined;
    });
  }
  var App = {};
  methods.forEach(function (m) {
    App[m] = function () { return rpc(m, Array.prototype.slice.call(arguments)); };
  });
  window.go = { main: { App: App } };

  var listeners = {};
  var source = null;
  function connect() {
    source = new EventSource('/api/events');
    source.onmessage = function (e) {
      var d;
      try { d = JSON.parse(e.data); } catch (_) { return; }
      (listeners[d.event] || []).forEach(function (cb) {
        try { cb(d.data); } catch (_) {}
      });
    };
  }
  window.runtime = {
    EventsOn: function (name, cb) {
      (listeners[name] = listeners[name] || []).push(cb);
      if (!source) { connect(); }
    },
    EventsOff: function (name) { delete listeners[name]; },
    Quit: function () {},
    WindowShow: function () {},
    WindowHide: function () {}
  };
})();`
