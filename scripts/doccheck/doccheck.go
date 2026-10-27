package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/PopGreenTh/OpenXOps/finops/analyze"
	"github.com/PopGreenTh/OpenXOps/finops/enrich"
	finopssense "github.com/PopGreenTh/OpenXOps/finops/sense"
	"github.com/PopGreenTh/OpenXOps/record"
	"github.com/PopGreenTh/OpenXOps/record/schema"
	xopsreport "github.com/PopGreenTh/OpenXOps/xops/report"
	xopssense "github.com/PopGreenTh/OpenXOps/xops/sense"
	"github.com/spf13/cobra"
)

// Run checks docs under root against the code.
func Run(root string) error {
	var errs []string
	errs = append(errs, checkRecords(root)...)
	errs = append(errs, checkFindings(root)...)
	errs = append(errs, checkCommands(root)...)
	errs = append(errs, checkLinks(root)...)
	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "\n"))
	}
	return nil
}

func checkRecords(root string) []string {
	path := filepath.Join(root, "docs/user/records.md")
	data, err := os.ReadFile(path)
	if err != nil {
		return []string{fmt.Sprintf("%s: %v", path, err)}
	}
	text := string(data)
	var errs []string
	for kind, sch := range schema.ByKind {
		sec := sectionForKind(text, kind)
		if sec == "" {
			errs = append(errs, fmt.Sprintf("%s: missing section for record kind %q", path, kind))
			continue
		}
		for _, f := range sch.Fields {
			if !fieldDocumented(sec, f.Name) {
				errs = append(errs, fmt.Sprintf("%s: kind %q missing field %q in records.md", path, kind, f.Name))
			}
		}
	}
	return errs
}

func sectionForKind(text, kind string) string {
	markers := []string{
		"## `" + kind + "`",
		"## " + kind,
	}
	for _, m := range markers {
		if i := strings.Index(text, m); i >= 0 {
			rest := text[i+len(m):]
			if j := strings.Index(rest, "\n## "); j >= 0 {
				return rest[:j]
			}
			return rest
		}
	}
	return ""
}

func fieldDocumented(section, field string) bool {
	// Table row: | `field` | or | field |
	for _, line := range strings.Split(section, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "|") {
			continue
		}
		if strings.Contains(line, "`"+field+"`") {
			return true
		}
		cells := splitTableRow(line)
		if len(cells) > 0 && strings.Trim(cells[0], " `") == field {
			return true
		}
	}
	return false
}

func splitTableRow(line string) []string {
	line = strings.Trim(line, "|")
	parts := strings.Split(line, "|")
	out := make([]string, len(parts))
	for i, p := range parts {
		out[i] = strings.TrimSpace(p)
	}
	return out
}

func checkFindings(root string) []string {
	path := filepath.Join(root, "docs/user/findings.md")
	data, err := os.ReadFile(path)
	if err != nil {
		return []string{fmt.Sprintf("%s: %v", path, err)}
	}
	docCodes := findingCodesInDoc(string(data))
	want := findingCodesInRepo(root)
	var errs []string
	for code := range want {
		if !docCodes[code] {
			errs = append(errs, fmt.Sprintf("%s: missing finding code %q", path, code))
		}
	}
	return errs
}

var findingCodeRe = regexp.MustCompile(`\|\s*` + "`" + `([a-z0-9-]+)` + "`" + `\s*\|`)

func findingCodesInDoc(text string) map[string]bool {
	out := map[string]bool{}
	for _, m := range findingCodeRe.FindAllStringSubmatch(text, -1) {
		out[m[1]] = true
	}
	return out
}

func findingCodesInRepo(root string) map[string]bool {
	absRoot, _ := filepath.Abs(root)
	inFixtureRun := strings.Contains(absRoot, filepath.Join("scripts", "doccheck", "testdata"))
	out := map[string]bool{}
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "vendor" || d.Name() == ".git" || d.Name() == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		if strings.Contains(path, filepath.Join("scripts", "doccheck")) &&
			!strings.Contains(path, filepath.Join("scripts", "doccheck", "testdata")) {
			return nil
		}
		if !inFixtureRun && strings.Contains(path, filepath.Join("scripts", "doccheck", "testdata")) {
			return nil
		}
		codes, err := findingLiteralsInFile(path)
		if err != nil {
			return err
		}
		for c := range codes {
			out[c] = true
		}
		return nil
	})
	return out
}

func findingLiteralsInFile(path string) (map[string]bool, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return nil, err
	}
	out := map[string]bool{}
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		collectFindingCodesInBlock(fn.Body, out)
	}
	return out, nil
}

func collectFindingCodesInBlock(body *ast.BlockStmt, out map[string]bool) {
	if body == nil {
		return
	}
	helpers := map[string]struct{}{}
	for _, stmt := range body.List {
		assign, ok := stmt.(*ast.AssignStmt)
		if !ok {
			continue
		}
		for i, rhs := range assign.Rhs {
			fl, ok := rhs.(*ast.FuncLit)
			if !ok || !forwardsCodeToFinding(fl) {
				continue
			}
			if id, ok := assign.Lhs[i].(*ast.Ident); ok {
				helpers[id.Name] = struct{}{}
			}
		}
	}
	ast.Inspect(body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.FuncLit:
			collectFindingCodesInBlock(node.Body, out)
			return false
		case *ast.CallExpr:
			if sel, ok := node.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "Finding" {
				if len(node.Args) >= 2 {
					if lit, ok := node.Args[1].(*ast.BasicLit); ok && lit.Kind == token.STRING {
						out[strings.Trim(lit.Value, `"`)] = true
					}
				}
			}
			if id, ok := node.Fun.(*ast.Ident); ok {
				if _, ok := helpers[id.Name]; ok && len(node.Args) >= 1 {
					if lit, ok := node.Args[0].(*ast.BasicLit); ok && lit.Kind == token.STRING {
						out[strings.Trim(lit.Value, `"`)] = true
					}
				}
			}
		}
		return true
	})
}

func forwardsCodeToFinding(fl *ast.FuncLit) bool {
	if fl.Type.Params.NumFields() < 1 || len(fl.Type.Params.List[0].Names) < 1 {
		return false
	}
	codeName := fl.Type.Params.List[0].Names[0].Name
	found := false
	ast.Inspect(fl.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Finding" || len(call.Args) < 2 {
			return true
		}
		if id, ok := call.Args[1].(*ast.Ident); ok && id.Name == codeName {
			found = true
			return false
		}
		return true
	})
	return found
}

func checkCommands(root string) []string {
	path := filepath.Join(root, "docs/user/commands.md")
	data, err := os.ReadFile(path)
	if err != nil {
		return []string{fmt.Sprintf("%s: %v", path, err)}
	}
	text := string(data)
	var errs []string
	for _, id := range registeredCommands() {
		if !commandDocumented(text, id) {
			errs = append(errs, fmt.Sprintf("%s: missing section for command %q", path, id))
		}
	}
	return errs
}

func registeredCommands() []string {
	var out []string
	xops := record.Root("xops", "")
	xops.AddCommand(xopssense.Commands()...)
	xops.AddCommand(xopsreport.Commands()...)
	out = append(out, leafCommandPaths(xops)...)

	finops := record.Root("finops", "")
	finops.AddCommand(finopssense.Commands()...)
	finops.AddCommand(enrich.Commands()...)
	finops.AddCommand(analyze.Commands()...)
	out = append(out, leafCommandPaths(finops)...)
	return out
}

func leafCommandPaths(root *cobra.Command) []string {
	var out []string
	var walk func(*cobra.Command)
	walk = func(c *cobra.Command) {
		if len(c.Commands()) == 0 {
			out = append(out, c.CommandPath())
			return
		}
		for _, sub := range c.Commands() {
			if sub.Hidden {
				continue
			}
			walk(sub)
		}
	}
	for _, sub := range root.Commands() {
		walk(sub)
	}
	return out
}

func commandDocumented(text, id string) bool {
	markers := []string{
		"## `" + id + "`",
		"## " + id,
	}
	for _, m := range markers {
		if strings.Contains(text, m) {
			return true
		}
	}
	return false
}

var mdLinkRe = regexp.MustCompile(`\]\(([^)]+)\)`)

func checkLinks(root string) []string {
	var files []string
	for _, f := range []string{"README.md", "CONTRIBUTING.md"} {
		p := filepath.Join(root, f)
		if _, err := os.Stat(p); err == nil {
			files = append(files, p)
		}
	}
	_ = filepath.WalkDir(filepath.Join(root, "docs"), func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".md") {
			files = append(files, path)
		}
		return nil
	})
	var errs []string
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", path, err))
			continue
		}
		dir := filepath.Dir(path)
		for _, m := range mdLinkRe.FindAllStringSubmatch(string(data), -1) {
			target := strings.TrimSpace(m[1])
			if target == "" || strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") || strings.HasPrefix(target, "#") {
				continue
			}
			target = strings.Split(target, "#")[0]
			if target == "" {
				continue
			}
			resolved := filepath.Clean(filepath.Join(dir, target))
			if _, err := os.Stat(resolved); err != nil {
				errs = append(errs, fmt.Sprintf("%s: broken link %q -> %s", path, target, resolved))
			}
		}
	}
	return errs
}
