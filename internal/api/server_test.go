package api

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

type fakeIn struct {
	Name string `json:"name"`
}

type fakeOut struct {
	Greeting string `json:"greeting"`
}

type fake struct{}

func (fake) Echo(s string) (string, error) {
	if s == "bad" {
		return "", errors.New("gagal sengaja")
	}
	return s + "!", nil
}

func (fake) Add(a, b int) int { return a + b }

func (fake) Fail() error { return errors.New("selalu gagal") }

func (fake) Greet(in fakeIn) (fakeOut, error) { return fakeOut{Greeting: "hai " + in.Name}, nil }

func (fake) NoReturn() {}

func newTestServer(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	assets := fstest.MapFS{
		"index.html":      {Data: []byte("<!doctype html><html><head><title>t</title></head><body><script type=\"module\" src=\"/assets/app.js\"></script></body></html>")},
		"assets/app.js":   {Data: []byte("console.log(1)")},
		"assets/app.css":  {Data: []byte("body{}")},
		"assets/logo.svg": {Data: []byte("<svg></svg>")},
	}
	srv, err := New(Options{Target: &fake{}, Assets: assets})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return srv, ts
}

func call(t *testing.T, ts *httptest.Server, method string, args ...any) callResponse {
	t.Helper()
	body, _ := json.Marshal(callRequest{Method: method, Args: rawArgs(t, args)})
	resp, err := http.Post(ts.URL+"/api/call", "application/json", strings.NewReader(string(body)))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out callResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	return out
}

func rawArgs(t *testing.T, args []any) []json.RawMessage {
	t.Helper()
	out := make([]json.RawMessage, len(args))
	for i, a := range args {
		b, err := json.Marshal(a)
		if err != nil {
			t.Fatal(err)
		}
		out[i] = b
	}
	return out
}

func TestCallSuccess(t *testing.T) {
	_, ts := newTestServer(t)
	if got := call(t, ts, "Echo", "hi"); got.Error != "" || got.Result != "hi!" {
		t.Fatalf("Echo = %+v", got)
	}
	if got := call(t, ts, "Add", 2, 40); got.Error != "" || got.Result != float64(42) {
		t.Fatalf("Add = %+v", got)
	}
	got := call(t, ts, "Greet", map[string]string{"name": "Ana"})
	if got.Error != "" {
		t.Fatalf("Greet error: %s", got.Error)
	}
	m, ok := got.Result.(map[string]any)
	if !ok || m["greeting"] != "hai Ana" {
		t.Fatalf("Greet = %+v", got.Result)
	}
}

func TestCallErrors(t *testing.T) {
	_, ts := newTestServer(t)
	if got := call(t, ts, "Echo", "bad"); got.Error != "gagal sengaja" {
		t.Fatalf("expected method error, got %+v", got)
	}
	if got := call(t, ts, "Fail"); got.Error != "selalu gagal" {
		t.Fatalf("expected error, got %+v", got)
	}
	if got := call(t, ts, "NoReturn"); got.Error != "" || got.Result != nil {
		t.Fatalf("NoReturn = %+v", got)
	}
	if got := call(t, ts, "Nope"); !strings.Contains(got.Error, "tidak dikenal") {
		t.Fatalf("unknown method: %+v", got)
	}
	if got := call(t, ts, "Echo"); !strings.Contains(got.Error, "butuh 1 argumen") {
		t.Fatalf("arg count: %+v", got)
	}
	// arg tipe salah
	body := `{"method":"Add","args":["x",2]}`
	resp, err := http.Post(ts.URL+"/api/call", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out callResponse
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if !strings.Contains(out.Error, "Argumen 1") {
		t.Fatalf("bad arg: %+v", out)
	}
}

func TestStaticAndShim(t *testing.T) {
	_, ts := newTestServer(t)
	index, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer index.Body.Close()
	buf, _ := io.ReadAll(index.Body)
	if !strings.Contains(string(buf), `/__devlok_shim.js`) {
		t.Fatalf("index tidak menyisipkan shim: %s", string(buf))
	}
	shim, err := http.Get(ts.URL + "/__devlok_shim.js")
	if err != nil {
		t.Fatal(err)
	}
	defer shim.Body.Close()
	sb, _ := io.ReadAll(shim.Body)
	if !strings.Contains(string(sb), `"Echo"`) || !strings.Contains(string(sb), "window.go") {
		t.Fatalf("shim tidak memuat method: %s", string(sb))
	}
	// SPA fallback untuk path tak dikenal
	spa, err := http.Get(ts.URL + "/some/deep/route")
	if err != nil {
		t.Fatal(err)
	}
	defer spa.Body.Close()
	if spa.StatusCode != http.StatusOK {
		t.Fatalf("SPA fallback status = %d", spa.StatusCode)
	}
	// aset statis
	js, err := http.Get(ts.URL + "/assets/app.js")
	if err != nil {
		t.Fatal(err)
	}
	defer js.Body.Close()
	if ct := js.Header.Get("Content-Type"); !strings.Contains(ct, "javascript") {
		t.Fatalf("content-type js = %q", ct)
	}
}

func TestEventsSSE(t *testing.T) {
	srv, ts := newTestServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL+"/api/events", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	lines := make(chan string, 4)
	go func() {
		sc := bufio.NewScanner(resp.Body)
		for sc.Scan() {
			if strings.HasPrefix(sc.Text(), "data: ") {
				lines <- strings.TrimPrefix(sc.Text(), "data: ")
			}
		}
	}()
	time.Sleep(100 * time.Millisecond)
	srv.Emit("update:progress", map[string]any{"percent": 50})
	select {
	case line := <-lines:
		var ev struct {
			Event string         `json:"event"`
			Data  map[string]any `json:"data"`
		}
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			t.Fatal(err)
		}
		if ev.Event != "update:progress" || ev.Data["percent"] != float64(50) {
			t.Fatalf("event = %+v", ev)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("tidak menerima event SSE")
	}
}
