package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type rpcDef struct {
	Method string
	Req    string
	Res    string
}

func main() {
	var idlDir, outDir string
	flag.StringVar(&idlDir, "idl-dir", "examples", "Directory to scan for IDL files (*.wb.idl)")
	flag.StringVar(&outDir, "out-dir", "pkg/wellsrpc/codec_generated", "Output directory")
	flag.Parse()

	files := []string{}
	err := filepath.Walk(idlDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".wb.idl") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		fmt.Println("scan idl-dir:", err)
		os.Exit(1)
	}

	for _, f := range files {
		if err := generateService(f, outDir); err != nil {
			fmt.Println("failed:", f, err)
		}
	}
}

func generateService(idlPath, outBase string) error {
	data, err := os.ReadFile(idlPath)
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	var srvName string
	rpcs := []rpcDef{}
	serviceRe := regexp.MustCompile(`^service\s+(\w+)`)
	rpcRe := regexp.MustCompile(`rpc\s+(\w+)\s*\(\s*(\w+)\s*\)\s*returns\s*\(\s*(\w+)\s*\)`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if m := serviceRe.FindStringSubmatch(line); m != nil {
			srvName = m[1]
		}
		if m := rpcRe.FindStringSubmatch(line); m != nil {
			rpcs = append(rpcs, rpcDef{Method: m[1], Req: m[2], Res: m[3]})
		}
	}

	if srvName == "" || len(rpcs) == 0 {
		return fmt.Errorf("no service or rpc in %s", idlPath)
	}

	pkgDir := filepath.Join(outBase, strings.ToLower(srvName))
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		return err
	}

	outFile := filepath.Join(pkgDir, strings.ToLower(srvName)+".pb.go")
	f, err := os.Create(outFile)
	if err != nil {
		return err
	}
	defer f.Close()

	writeHeader(f, srvName)
	writeServer(f, srvName, rpcs)
	writeClient(f, srvName, rpcs)
	writeHelper(f, srvName, rpcs)

	fmt.Println("Generated:", outFile)
	return nil
}

func writeHeader(f *os.File, srvName string) {
	fmt.Fprintln(f, "package", strings.ToLower(srvName))
	fmt.Fprintln(f, `import (`)
	fmt.Fprintln(f, `"context"`)
	fmt.Fprintln(f, `wellib "github.com/welliardiansyah/wells-rpc/pkg/wellsrpc"`)
	fmt.Fprintln(f, ")")
	fmt.Fprintln(f, "")
}

func writeServer(f *os.File, srvName string, rpcs []rpcDef) {
	fmt.Fprintf(f, "type %sServer interface {", srvName)
	for _, r := range rpcs {
		fmt.Fprintf(f, "  %s(ctx context.Context, req *%s) (*%s, error)", r.Method, r.Req, r.Res)
	}
	fmt.Fprintln(f, "}")

	fmt.Fprintf(f, "func NewServer(impl %sServer) *wellib.RPCServer {", srvName)
	fmt.Fprintln(f, "  srv := wellib.NewRPCServer()")
	for _, r := range rpcs {
		fmt.Fprintf(f, "  srv.Register(\"%s.%s\", func(ctx context.Context, payload []byte) ([]byte, error) {", srvName, r.Method)
		fmt.Fprintf(f, "    var req %s", r.Req)
		fmt.Fprintln(f, "    if err := req.UnmarshalWells(payload); err != nil { return nil, err }")
		fmt.Fprintf(f, "    resp, err := impl.%s(ctx, &req)", r.Method)
		fmt.Fprintln(f, "    if err != nil { return nil, err }")
		fmt.Fprintln(f, "    return resp.MarshalWells(), nil")
		fmt.Fprintln(f, "  })")
	}
	fmt.Fprintln(f, "  return srv}")
	fmt.Fprintln(f, "")
}

func writeClient(f *os.File, srvName string, rpcs []rpcDef) {
	fmt.Fprintf(f, "type Client struct { c *wellib.RPCClient }")
	fmt.Fprintf(f, "func NewClient(addr string) *Client {")
	fmt.Fprintf(f, "  conn, _ := wellib.Dial(addr, nil)")
	fmt.Fprintf(f, "  return &Client{c: conn}")
	fmt.Fprintln(f, "}")

	for _, r := range rpcs {
		fmt.Fprintf(f, "func (c *Client) %s(ctx context.Context, req *%s) (*%s, error) {", r.Method, r.Req, r.Res)
		fmt.Fprintf(f, "  var out %s", r.Res)
		fmt.Fprintf(f, "  if err := c.c.Call(ctx, \"%s.%s\", req, &out); err != nil { return nil, err }", srvName, r.Method)
		fmt.Fprintln(f, "  return &out, nil")
		fmt.Fprintln(f, "}")
	}
}

func writeHelper(f *os.File, srvName string, rpcs []rpcDef) {
	fmt.Fprintf(f, "// High-level simple client")
	fmt.Fprintf(f, "type SimpleClient struct { client *Client }")
	fmt.Fprintf(f, "func NewSimpleClient(addr string) *SimpleClient {")
	fmt.Fprintf(f, "  return &SimpleClient{client: NewClient(addr)}")
	fmt.Fprintln(f, "}")

	for _, r := range rpcs {
		fmt.Fprintf(f, "func (s *SimpleClient) %s(ctx context.Context, req *%s) (*%s, error) {", r.Method, r.Req, r.Res)
		fmt.Fprintln(f, "  return s.client."+r.Method+"(ctx, req)")
		fmt.Fprintln(f, "}")
	}
}
