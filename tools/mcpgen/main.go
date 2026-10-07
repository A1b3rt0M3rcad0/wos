package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
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
	type entry struct{ name, cmd, result string }
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
				var result bytes.Buffer
				rt := fn.Type.Results.List[0].Type.(*ast.IndexExpr)
				printer.Fprint(&result, fset, rt.Index)
				resultType := result.String()
				if !strings.Contains(resultType, ".") {
					resultType = "application." + resultType
				}
				entries = append(entries, entry{fn.Name.Name, cmd.Name, resultType})
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
	web, err := os.Create("packages/wos-api/commands/catalog_generated.go")
	if err != nil {
		panic(err)
	}
	defer web.Close()
	fmt.Fprintln(web, "// Code generated from public Application commands; DO NOT EDIT.\npackage commands\nimport(\"context\";\"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application\";\"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain\";\"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports\")\nfunc NewCatalog(service *application.Service,ids ports.IDGenerator)*Catalog {c:=newCatalog(ids)")
	for _, e := range entries {
		fmt.Fprintf(web, "register[application.%s](c,\"%s\",func(ctx context.Context,cc domain.CommandContext,cmd application.%s)(any,error){return service.%s(ctx,cc,cmd)})\n", e.cmd, snake(e.name), e.cmd, e.name)
	}
	fmt.Fprintln(web, "return c}")

	sdk, err := os.Create("packages/wos-sdk-go/commands_generated.go")
	if err != nil {
		panic(err)
	}
	defer sdk.Close()
	fmt.Fprintln(sdk, "// Code generated from public Application commands; DO NOT EDIT.\npackage wossdk\nimport(\"context\";\"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application\";\"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain\")")
	for _, e := range entries {
		fmt.Fprintf(sdk, "func(c *Client)%s(ctx context.Context,key string,cmd application.%s)(CommandResult[%s],error){return command[%s](ctx,c,\"%s\",key,cmd)}\n", e.name, e.cmd, e.result, e.result, snake(e.name))
	}

	fmt.Println("registered", len(entries), "commands")
}
