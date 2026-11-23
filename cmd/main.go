// Command myparser parses one SQL statement and reports its AST, dialect
// compatibility issues and warnings. It is a thin CLI over the public API.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mia0x75/myparser"
	"github.com/mia0x75/myparser/ast"
	"github.com/mia0x75/myparser/format"
)

// result is the stable JSON schema printed with -json.
type result struct {
	SQL        string   `json:"sql"`
	Dialect    string   `json:"dialect"`
	Restored   []string `json:"restored,omitempty"`
	Warnings   []string `json:"warnings,omitempty"`
	Issues     []string `json:"issues,omitempty"`
	ParseError string   `json:"parse_error,omitempty"`
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("myparser", flag.ContinueOnError)
	fs.SetOutput(stderr)
	sqlFlag := fs.String("sql", "", "SQL statement to parse")
	dialectFlag := fs.String("dialect", "union", "target dialect: union, mysql or mariadb")
	jsonFlag := fs.Bool("json", false, "emit a JSON result instead of restored SQL")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	text := strings.TrimSpace(*sqlFlag)
	if text == "" && fs.NArg() > 0 {
		text = strings.TrimSpace(strings.Join(fs.Args(), " "))
	}
	if text == "" && stdin != nil {
		data, err := io.ReadAll(stdin)
		if err == nil {
			text = strings.TrimSpace(string(data))
		}
	}
	if text == "" {
		_, _ = fmt.Fprintln(stderr, "provide SQL via -sql, a positional argument, or stdin")
		return 2
	}

	target, err := parseDialect(*dialectFlag)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 2
	}

	p := myparser.New()
	p.EnableWindowFunc(true)
	res := result{SQL: text, Dialect: target.String()}

	var stmts []ast.StmtNode
	var warnings []error
	if target == myparser.DialectUnion {
		stmts, warnings, err = p.Parse(text, "", "")
	} else {
		var stmt ast.StmtNode
		stmt, err = p.ParseOneStmtForDialect(text, "", "", target)
		if err == nil && stmt != nil {
			stmts = []ast.StmtNode{stmt}
		}
		// Strict parsing collects compatibility issues into a typed error;
		// surface them as a list instead of a single multi-line message.
		var compat *myparser.DialectCompatibilityError
		if errors.As(err, &compat) {
			for _, issue := range compat.Issues {
				res.Issues = append(res.Issues, issue.Error())
			}
			err = nil
		}
	}
	if err != nil {
		res.ParseError = err.Error()
		writeResult(stdout, stderr, res, *jsonFlag)
		return 1
	}
	for _, warning := range warnings {
		res.Warnings = append(res.Warnings, warning.Error())
	}
	for _, stmt := range stmts {
		var buf strings.Builder
		if err := stmt.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &buf)); err != nil {
			_, _ = fmt.Fprintln(stderr, err)
			return 1
		}
		res.Restored = append(res.Restored, buf.String())
	}
	writeResult(stdout, stderr, res, *jsonFlag)
	if len(res.Issues) > 0 {
		return 3 // parsed, but rejected by the strict dialect
	}
	return 0
}

func parseDialect(name string) (myparser.Dialect, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "union":
		return myparser.DialectUnion, nil
	case "mysql":
		return myparser.DialectMySQL, nil
	case "mariadb":
		return myparser.DialectMariaDB, nil
	default:
		return myparser.DialectUnion, fmt.Errorf("unsupported dialect %q: expected union, mysql or mariadb", name)
	}
}

func writeResult(stdout, stderr io.Writer, res result, asJSON bool) {
	if !asJSON {
		for _, issue := range res.Issues {
			_, _ = fmt.Fprintln(stderr, "issue:", issue)
		}
		for _, w := range res.Warnings {
			_, _ = fmt.Fprintln(stderr, "warning:", w)
		}
		if res.ParseError != "" {
			_, _ = fmt.Fprintln(stderr, "error:", res.ParseError)
		}
		for _, sql := range res.Restored {
			_, _ = fmt.Fprintln(stdout, sql)
		}
		return
	}
	data, err := json.Marshal(res)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return
	}
	_, _ = fmt.Fprintln(stdout, string(data))
}
