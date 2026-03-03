package confetti

import (
	"reflect"
	"testing"
)

// decodeOK parses src then decodes into v; fails the test on any error.
func decodeOK(t *testing.T, src string, v any) {
	t.Helper()
	if err := Unmarshal(src, v); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
}

// Test 1: Simple scalar fields — string, int, bool, float64
func TestDecode_SimpleScalars(t *testing.T) {
	type Config struct {
		Host    string  `conf:"host"`
		Port    int     `conf:"port"`
		Debug   bool    `conf:"debug"`
		Ratio   float64 `conf:"ratio"`
	}
	src := "host example.com\nport 8080\ndebug true\nratio 1.5\n"
	var got Config
	decodeOK(t, src, &got)
	want := Config{Host: "example.com", Port: 8080, Debug: true, Ratio: 1.5}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

// Test 2: Multi-value field — []string from extra args
func TestDecode_StringSlice(t *testing.T) {
	type Config struct {
		Tags []string `conf:"tags"`
	}
	var got Config
	decodeOK(t, "tags foo bar baz\n", &got)
	want := Config{Tags: []string{"foo", "bar", "baz"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

// Test 3: Nested struct via block directive
func TestDecode_NestedStruct(t *testing.T) {
	type TLS struct {
		Cert string `conf:"cert"`
		Key  string `conf:"key"`
	}
	type Config struct {
		TLS TLS `conf:"tls"`
	}
	src := "tls {\n  cert /etc/cert.pem\n  key /etc/key.pem\n}\n"
	var got Config
	decodeOK(t, src, &got)
	want := Config{TLS: TLS{Cert: "/etc/cert.pem", Key: "/etc/key.pem"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

// Test 4: Inline arg (,arg) on block directive
func TestDecode_ArgField(t *testing.T) {
	type Server struct {
		Name    string `conf:",arg"`
		Timeout int    `conf:"timeout"`
	}
	type Config struct {
		Server Server `conf:"server"`
	}
	src := "server web {\n  timeout 30\n}\n"
	var got Config
	decodeOK(t, src, &got)
	want := Config{Server: Server{Name: "web", Timeout: 30}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

// Test 5: Repeated block directives → []Struct
func TestDecode_SliceOfStructs(t *testing.T) {
	type Server struct {
		Name    string `conf:",arg"`
		Timeout int    `conf:"timeout"`
	}
	type Config struct {
		Servers []Server `conf:"server"`
	}
	src := "server web {\n  timeout 30\n}\nserver api {\n  timeout 60\n}\n"
	var got Config
	decodeOK(t, src, &got)
	want := Config{Servers: []Server{
		{Name: "web", Timeout: 30},
		{Name: "api", Timeout: 60},
	}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

// Test 6: Tag name override
func TestDecode_TagNameOverride(t *testing.T) {
	type Config struct {
		MaxConn int `conf:"max-connections"`
	}
	var got Config
	decodeOK(t, "max-connections 100\n", &got)
	if got.MaxConn != 100 {
		t.Fatalf("got MaxConn=%d, want 100", got.MaxConn)
	}
}

// Test 7: "-" tag → field is skipped
func TestDecode_DashTag_Skipped(t *testing.T) {
	type Config struct {
		Host    string `conf:"host"`
		Ignored string `conf:"-"`
	}
	var got Config
	decodeOK(t, "host example.com\nignored should-not-appear\n", &got)
	if got.Host != "example.com" {
		t.Fatalf("got Host=%q, want example.com", got.Host)
	}
	if got.Ignored != "" {
		t.Fatalf("Ignored field should remain zero, got %q", got.Ignored)
	}
}

// Test 8: Unknown directives → silently ignored
func TestDecode_UnknownDirectives_Ignored(t *testing.T) {
	type Config struct {
		Host string `conf:"host"`
	}
	var got Config
	decodeOK(t, "host example.com\nunknown something\n", &got)
	if got.Host != "example.com" {
		t.Fatalf("got Host=%q, want example.com", got.Host)
	}
}

// Test 9: Type coercion error → returns error
func TestDecode_TypeError(t *testing.T) {
	type Config struct {
		Port int `conf:"port"`
	}
	var got Config
	if err := Unmarshal("port abc\n", &got); err == nil {
		t.Fatal("expected error for port=abc, got nil")
	}
}

// Test 10: Unmarshal end-to-end convenience wrapper
func TestDecode_Unmarshal_EndToEnd(t *testing.T) {
	type Config struct {
		Host    string   `conf:"host"`
		Port    int      `conf:"port"`
		Debug   bool     `conf:"debug"`
		Tags    []string `conf:"tags"`
		Servers []struct {
			Name    string `conf:",arg"`
			Timeout int    `conf:"timeout"`
		} `conf:"server"`
	}
	src := `
host example.com
port 8080
debug true
tags foo bar baz

server web {
    timeout 30
}
server api {
    timeout 60
}
`
	var got Config
	if err := Unmarshal(src, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.Host != "example.com" {
		t.Errorf("Host: got %q", got.Host)
	}
	if got.Port != 8080 {
		t.Errorf("Port: got %d", got.Port)
	}
	if !got.Debug {
		t.Errorf("Debug: got false")
	}
	if !reflect.DeepEqual(got.Tags, []string{"foo", "bar", "baz"}) {
		t.Errorf("Tags: got %v", got.Tags)
	}
	if len(got.Servers) != 2 {
		t.Fatalf("Servers: got %d elements, want 2", len(got.Servers))
	}
	if got.Servers[0].Name != "web" || got.Servers[0].Timeout != 30 {
		t.Errorf("Servers[0]: got %+v", got.Servers[0])
	}
	if got.Servers[1].Name != "api" || got.Servers[1].Timeout != 60 {
		t.Errorf("Servers[1]: got %+v", got.Servers[1])
	}
}

// Test 11: Pointer to struct field (*Server)
func TestDecode_PointerToStruct(t *testing.T) {
	type Server struct {
		Name    string `conf:",arg"`
		Timeout int    `conf:"timeout"`
	}
	type Config struct {
		Server *Server `conf:"server"`
	}
	src := "server web {\n  timeout 30\n}\n"
	var got Config
	decodeOK(t, src, &got)
	if got.Server == nil {
		t.Fatal("Server pointer is nil")
	}
	want := &Server{Name: "web", Timeout: 30}
	if !reflect.DeepEqual(got.Server, want) {
		t.Fatalf("got %+v, want %+v", got.Server, want)
	}
}

// Test 12: Decode with non-pointer input → error
func TestDecode_NonPointer_Error(t *testing.T) {
	type Config struct {
		Host string `conf:"host"`
	}
	cfg := &ConfigurationUnit{}
	if err := Decode(cfg, Config{}); err == nil {
		t.Fatal("expected error for non-pointer v, got nil")
	}
}

// Test 12b: []*Struct slice
func TestDecode_SliceOfPointerToStruct(t *testing.T) {
	type Server struct {
		Name    string `conf:",arg"`
		Timeout int    `conf:"timeout"`
	}
	type Config struct {
		Servers []*Server `conf:"server"`
	}
	src := "server web {\n  timeout 30\n}\nserver api {\n  timeout 60\n}\n"
	var got Config
	decodeOK(t, src, &got)
	if len(got.Servers) != 2 {
		t.Fatalf("want 2 servers, got %d", len(got.Servers))
	}
	if got.Servers[0].Name != "web" || got.Servers[0].Timeout != 30 {
		t.Errorf("Servers[0]: got %+v", got.Servers[0])
	}
	if got.Servers[1].Name != "api" || got.Servers[1].Timeout != 60 {
		t.Errorf("Servers[1]: got %+v", got.Servers[1])
	}
}

// Implicit field name (no tag) → lowercase field name used
func TestDecode_ImplicitFieldName(t *testing.T) {
	type Config struct {
		Host string
		Port int
	}
	var got Config
	decodeOK(t, "host localhost\nport 9090\n", &got)
	want := Config{Host: "localhost", Port: 9090}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}
