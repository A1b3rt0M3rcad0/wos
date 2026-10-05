package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"strings"
	"unicode"
)

func snake(s string) string {
	r := []rune(s)
	var b strings.Builder
	for i, c := range r {
		if unicode.IsUpper(c) && i > 0 && (unicode.IsLower(r[i-1]) || (i+1 < len(r) && unicode.IsLower(r[i+1]))) {
			b.WriteByte('_')
		}
		b.WriteRune(unicode.ToLower(c))
	}
	return b.String()
}
func main() {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, "packages/wos-core/application", func(f os.FileInfo) bool { return !strings.HasSuffix(f.Name(), "_test.go") }, 0)
	if err != nil {
		panic(err)
	}
	type entry struct{ name, cmd string }
	var entries []entry
	for _, p := range pkgs {
		for _, f := range p.Files {
			for _, d := range f.Decls {
				fn, ok := d.(*ast.FuncDecl)
				if !ok || fn.Recv == nil || !fn.Name.IsExported() || len(fn.Type.Params.List) != 3 {
					continue
				}
				recv, ok := fn.Recv.List[0].Type.(*ast.StarExpr)
				if !ok {
					continue
				}
				typ, ok := recv.X.(*ast.Ident)
				if !ok || typ.Name != "Service" {
					continue
				}
				cc, ok := fn.Type.Params.List[1].Type.(*ast.SelectorExpr)
				if !ok || cc.Sel.Name != "CommandContext" {
					continue
				}
				cmd, ok := fn.Type.Params.List[2].Type.(*ast.Ident)
				if !ok || !strings.HasSuffix(cmd.Name, "Command") {
					continue
				}
				entries = append(entries, entry{fn.Name.Name, cmd.Name})
			}
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].name < entries[j].name })
	out, err := os.Create("packages/wos-api/mcp/commands_generated.go")
	if err != nil {
		panic(err)
	}
	defer out.Close()
	fmt.Fprintln(out, "// Code generated from public Application command signatures; DO NOT EDIT.\npackage mcptransport\nimport(\"context\";\"github.com/modelcontextprotocol/go-sdk/mcp\";\"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application\";\"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain\";\"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports\")\nfunc registerCommands(server *mcp.Server, service *application.Service, ids ports.IDGenerator, options Options) {")
	for _, e := range entries {
		fmt.Fprintf(out, "registerCommand[application.%s](server,ids,options,\"wos_%s\",func(ctx context.Context,cc domain.CommandContext,cmd application.%s)(any,error){return service.%s(ctx,cc,cmd)})\n", e.cmd, snake(e.name), e.cmd, e.name)
	}
	fmt.Fprintln(out, "}")
	fmt.Println("registered", len(entries), "commands")
}
