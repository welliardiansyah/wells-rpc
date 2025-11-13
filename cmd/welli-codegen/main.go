package main

import (
	"bufio"
	"flag"
	"fmt"
	"go/format"
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

type messageDef struct {
	Name   string
	Fields []fieldDef
}

type fieldDef struct {
	Name string
	Type string
	Tag  int
}

func main() {
	var (
		idlPath  string
		outDir   string
		showHelp bool
	)

	flag.StringVar(&idlPath, "idl", "", "Path to .wb.idl file or directory containing IDL files")
	flag.StringVar(&outDir, "out", "", "Output directory for generated Go code")
	flag.BoolVar(&showHelp, "help", false, "Show usage help")
	flag.BoolVar(&showHelp, "h", false, "Show usage help (shorthand)")
	flag.Parse()

	if showHelp {
		printHelp()
		return
	}

	if idlPath == "" || outDir == "" {
		printHelp()
		os.Exit(1)
	}

	info, err := os.Stat(idlPath)
	if err != nil {
		fmt.Println("❌ Error:", err)
		os.Exit(1)
	}

	var files []string
	if info.IsDir() {
		err := filepath.Walk(idlPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && strings.HasSuffix(info.Name(), ".wb.idl") {
				files = append(files, path)
			}
			return nil
		})
		if err != nil {
			fmt.Println("❌ Failed to scan directory:", err)
			os.Exit(1)
		}
	} else {
		if !strings.HasSuffix(info.Name(), ".wb.idl") {
			fmt.Println("❌ Error: IDL file must have .wb.idl extension")
			os.Exit(1)
		}
		files = append(files, idlPath)
	}

	if len(files) == 0 {
		fmt.Println("⚠️  No .wb.idl files found in:", idlPath)
		return
	}

	for _, f := range files {
		fmt.Printf("⚙️  Generating from %s...\n", f)
		if err := generateService(f, outDir); err != nil {
			fmt.Println("❌ Failed:", f, "error:", err)
		} else {
			fmt.Printf("✅ Successfully generated from %s\n", f)
		}
	}
}

func printHelp() {
	fmt.Print(`
WellsRPC Code Generator

Usage:
  welli-codegen -idl <path> -out <output_dir>

Examples:
  welli-codegen -idl ./idl/sensor.wb.idl -out ./wellsrpc
  welli-codegen -idl ./idl -out ./generated

Options:
  -idl        Path to .wb.idl file or directory containing IDL files
  -out        Output directory for generated Go code
  -h, --help  Show this help message
`)
}

func generateService(idlPath, outBase string) error {
	data, err := os.ReadFile(idlPath)
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	var srvName string
	rpcs := []rpcDef{}
	messages := []messageDef{}

	serviceRe := regexp.MustCompile(`^service\s+(\w+)`)
	rpcRe := regexp.MustCompile(`rpc\s+(\w+)\s*\(\s*(\w+)\s*\)\s*returns\s*\(\s*(\w+)\s*\)`)
	messageRe := regexp.MustCompile(`^message\s+(\w+)`)
	fieldRe := regexp.MustCompile(`^(\w+)\s+(\w+);`)

	var currentMsg *messageDef
	tagCounter := 1

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}

		if m := serviceRe.FindStringSubmatch(line); m != nil {
			srvName = m[1]
		}
		if m := rpcRe.FindStringSubmatch(line); m != nil {
			rpcs = append(rpcs, rpcDef{Method: m[1], Req: m[2], Res: m[3]})
		}
		if m := messageRe.FindStringSubmatch(line); m != nil {
			if currentMsg != nil {
				messages = append(messages, *currentMsg)
			}
			currentMsg = &messageDef{Name: m[1], Fields: []fieldDef{}}
			tagCounter = 1
			continue
		}
		if currentMsg != nil {
			if f := fieldRe.FindStringSubmatch(line); f != nil {
				currentMsg.Fields = append(currentMsg.Fields, fieldDef{
					Type: f[1],
					Name: f[2],
					Tag:  tagCounter,
				})
				tagCounter++
			}
		}
	}
	if currentMsg != nil {
		messages = append(messages, *currentMsg)
	}

	if srvName == "" || len(rpcs) == 0 {
		return fmt.Errorf("no valid service or rpc definition in %s", idlPath)
	}

	pkgDir := filepath.Join(outBase, strings.ToLower(srvName))
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		return err
	}

	if err := writeCodec(pkgDir, messages); err != nil {
		return err
	}
	if err := writeServer(pkgDir, srvName, rpcs); err != nil {
		return err
	}
	if err := writeClient(pkgDir, srvName, rpcs); err != nil {
		return err
	}

	fmt.Println("📦 Generated service:", srvName, "→", pkgDir)
	return nil
}

func writeCodec(pkgDir string, messages []messageDef) error {
	file := filepath.Join(pkgDir, "codec.go")
	f, err := os.Create(file)
	if err != nil {
		return err
	}
	defer f.Close()

	fmt.Fprintf(f, "package %s\n\n", filepath.Base(pkgDir))
	for _, msg := range messages {
		fmt.Fprintf(f, "\ntype %s struct {\n", msg.Name)
		for _, field := range msg.Fields {
			fmt.Fprintf(f, "  %s %s\n", strings.Title(field.Name), mapType(field.Type))
		}
		fmt.Fprintln(f, "}")

		fmt.Fprintf(f, "\nfunc (m *%s) MarshalWells() []byte { return nil }\n", msg.Name)
		fmt.Fprintf(f, "func (m *%s) UnmarshalWells(b []byte) error { return nil }\n", msg.Name)
	}

	return formatFile(f)
}

func writeServer(pkgDir, srvName string, rpcs []rpcDef) error {
	file := filepath.Join(pkgDir, "server.go")
	f, err := os.Create(file)
	if err != nil {
		return err
	}
	defer f.Close()

	fmt.Fprintf(f, "package %s\n\n", filepath.Base(pkgDir))
	fmt.Fprintln(f, `import (
  "context"
  wellib "github.com/welliardiansyah/wells-rpc/pkg/wellsrpc"
)`)

	fmt.Fprintf(f, "\ntype %sServer interface {\n", srvName)
	for _, r := range rpcs {
		fmt.Fprintf(f, "  %s(ctx context.Context, req *%s) (*%s, error)\n", r.Method, r.Req, r.Res)
	}
	fmt.Fprintln(f, "}")

	fmt.Fprintf(f, "\nfunc Register%sServer(srv *wellib.RPCServer, impl %sServer) {\n", srvName, srvName)
	for _, r := range rpcs {
		fmt.Fprintf(f, "  srv.Register(\"%s.%s\", func(ctx context.Context, payload []byte) ([]byte, error) {\n", srvName, r.Method)
		fmt.Fprintf(f, "    var req %s\n", r.Req)
		fmt.Fprintln(f, "    if err := req.UnmarshalWells(payload); err != nil { return nil, err }")
		fmt.Fprintf(f, "    resp, err := impl.%s(ctx, &req)\n", r.Method)
		fmt.Fprintln(f, "    if err != nil { return nil, err }")
		fmt.Fprintln(f, "    return resp.MarshalWells(), nil")
		fmt.Fprintln(f, "  })")
	}
	fmt.Fprintln(f, "}")

	return formatFile(f)
}

func writeClient(pkgDir, srvName string, rpcs []rpcDef) error {
	file := filepath.Join(pkgDir, "client.go")
	f, err := os.Create(file)
	if err != nil {
		return err
	}
	defer f.Close()

	fmt.Fprintf(f, "package %s\n\n", filepath.Base(pkgDir))
	fmt.Fprintln(f, `import (
  "context"
  wellib "github.com/welliardiansyah/wells-rpc/pkg/wellsrpc"
)`)

	fmt.Fprintf(f, "\ntype %sClient struct {\n  c *wellib.RPCClient\n}\n\n", srvName)
	fmt.Fprintf(f, "func New%sClient(addr string) *%sClient {\n", srvName, srvName)
	fmt.Fprintln(f, "  conn, _ := wellib.Dial(addr, nil)")
	fmt.Fprintf(f, "  return &%sClient{c: conn}\n}\n", srvName)

	for _, r := range rpcs {
		fmt.Fprintf(f, "\nfunc (c *%sClient) %s(ctx context.Context, req *%s) (*%s, error) {\n", srvName, r.Method, r.Req, r.Res)
		fmt.Fprintf(f, "  var out %s\n", r.Res)
		fmt.Fprintf(f, "  if err := c.c.Call(ctx, \"%s.%s\", req, &out); err != nil { return nil, err }\n", srvName, r.Method)
		fmt.Fprintln(f, "  return &out, nil")
		fmt.Fprintln(f, "}")
	}

	return formatFile(f)
}

func mapType(t string) string {
	switch t {
	case "int64", "int32", "float32", "float64", "string", "bool":
		return t
	case "bytes":
		return "[]byte"
	default:
		return "*" + t
	}
}

func formatFile(f *os.File) error {
	data, err := os.ReadFile(f.Name())
	if err != nil {
		return err
	}
	formatted, err := format.Source(data)
	if err != nil {
		return err
	}
	return os.WriteFile(f.Name(), formatted, 0644)
}
