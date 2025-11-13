package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: welli-codegen <idl_file> [out_dir]")
		os.Exit(1)
	}
	idl := os.Args[1]
	outDir := "pkg/wellsrpc/codec_generated"
	if len(os.Args) >= 3 {
		outDir = os.Args[2]
	}

	data, err := os.ReadFile(idl)
	if err != nil {
		fmt.Println("read idl:", err)
		os.Exit(1)
	}

	if err := os.MkdirAll(outDir, 0755); err != nil {
		fmt.Println("mkdir:", err)
		os.Exit(1)
	}

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	var srvName string
	var rpcs []rpcDef

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
		fmt.Println("no service or rpc found in IDL")
		os.Exit(1)
	}

	outFile := filepath.Join(outDir, strings.ToLower(srvName)+"_rpc_stub.go")
	f, err := os.Create(outFile)
	if err != nil {
		fmt.Println("create:", err)
		os.Exit(1)
	}
	defer f.Close()

	fmt.Fprintln(f, "package codec_generated")
	fmt.Fprintln(f)
	fmt.Fprintln(f, `import (`)
	fmt.Fprintln(f, `  "context"`)
	fmt.Fprintln(f, `  wellib "github.com/welliardiansyah/wells-rpc/pkg/wellsrpc"`)
	fmt.Fprintln(f, `)`)
	fmt.Fprintln(f)

	fmt.Fprintf(f, "type %sServer interface {\n", srvName)
	for _, r := range rpcs {
		fmt.Fprintf(f, "  %s(ctx context.Context, req *%s) (*%s, error)\n", r.Method, r.Req, r.Res)
	}
	fmt.Fprintln(f, "}")
	fmt.Fprintln(f)

	fmt.Fprintf(f, "func Register%s(srv *wellib.RPCServer, impl %sServer) {\n", srvName, srvName)
	for _, r := range rpcs {
		methodName := srvName + "." + r.Method
		fmt.Fprintf(f, "  srv.Register(\"%s\", func(ctx context.Context, payload []byte) ([]byte, error) {\n", methodName)
		fmt.Fprintf(f, "    var req %s\n", r.Req)
		fmt.Fprintln(f, "    if err := req.UnmarshalWelli(payload); err != nil { return nil, err }")
		fmt.Fprintf(f, "    resp, err := impl.%s(ctx, &req)\n", r.Method)
		fmt.Fprintln(f, "    if err != nil { return nil, err }")
		fmt.Fprintln(f, "    return resp.MarshalWelli(), nil")
		fmt.Fprintln(f, "  })")
	}
	fmt.Fprintln(f, "}")
	fmt.Fprintln(f)

	fmt.Fprintf(f, "type %sClient struct{\n  c *wellib.RPCClient\n}\n\n", srvName)
	fmt.Fprintf(f, "func New%sClient(c *wellib.RPCClient) *%sClient { return &%sClient{c:c} }\n\n", srvName, srvName, srvName)

	for _, r := range rpcs {
		fmt.Fprintf(f, "func (c *%sClient) %s(ctx context.Context, req *%s) (*%s, error) {\n", srvName, r.Method, r.Req, r.Res)
		fmt.Fprintf(f, "  var out %s\n", r.Res)
		fmt.Fprintf(f, "  if err := c.c.Call(ctx, \"%s.%s\", req, &out); err != nil {\n", srvName, r.Method)
		fmt.Fprintln(f, "    return nil, err")
		fmt.Fprintln(f, "  }")
		fmt.Fprintln(f, "  return &out, nil")
		fmt.Fprintln(f, "}")
		fmt.Fprintln(f)
	}

	fmt.Println("Generated stub:", outFile)
}

type rpcDef struct {
	Method string
	Req    string
	Res    string
}
