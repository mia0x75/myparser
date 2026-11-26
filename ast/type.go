package ast

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	pkast "github.com/mia0x75/parserkit/ast"
	"github.com/mia0x75/parserkit/lex"

	"github.com/mia0x75/myparser/errors"
	"github.com/mia0x75/myparser/format"
	"github.com/mia0x75/myparser/mysql"
)

// TypeName carries declared type syntax, not a resolved FieldType.
type TypeName struct {
	pkast.BaseTypeName
	Charset   lex.Ident
	Collation lex.Ident
	Unsigned  bool
	Zerofill  bool
	Binary    bool
}

func NewTypeName(keyword string) *TypeName {
	return &TypeName{BaseTypeName: pkast.BaseTypeName{Keywords: strings.Fields(strings.ToUpper(keyword))}}
}

func (n *TypeName) Keyword() string { return strings.Join(n.Keywords, " ") }

// TypeCore returns the embedded type syntax, or nil for an absent type.
func (n *TypeName) TypeCore() *pkast.BaseTypeName {
	if n == nil {
		return nil
	}
	return &n.BaseTypeName
}

func (n *TypeName) number(index int) (int, bool) {
	if index >= len(n.Modifiers) {
		return 0, false
	}
	literal, ok := n.Modifiers[index].(*LiteralExpr)
	if !ok || literal == nil || literal.Kind != pkast.LiteralInteger {
		return 0, false
	}
	value, err := strconv.Atoi(literal.Value)
	return value, err == nil
}

func (n *TypeName) restoreKeyword() string {
	keyword := n.Keyword()
	switch keyword {
	case "BOOL", "BOOLEAN":
		return "TINYINT"
	case "UUID":
		return "CHAR"
	case "INET4":
		return "BIGINT"
	case "INET6":
		return "VARBINARY"
	case "FLOAT":
		if width, ok := n.number(0); ok && width > 24 {
			return "DOUBLE"
		}
	}
	if strings.EqualFold(n.Charset.Name, "binary") {
		switch keyword {
		case "CHAR":
			return "BINARY"
		case "VARCHAR":
			return "VARBINARY"
		case "TINYTEXT", "TEXT", "MEDIUMTEXT", "LONGTEXT":
			return strings.Replace(keyword, "TEXT", "BLOB", 1)
		}
	}
	return keyword
}

func (n *TypeName) validateKeyword() error {
	if len(n.Name.Parts) != 0 {
		if len(n.Keywords) != 0 {
			return errors.New("TypeName cannot contain both a qualified name and keywords")
		}
		return nil
	}
	if n.Keyword() == "" {
		return nil
	}
	switch strings.ToUpper(n.Keyword()) {
	case "BOOL", "BOOLEAN", "UUID", "INET4", "INET6", "FLOAT", "DOUBLE", "CHAR", "VARCHAR",
		"BINARY", "VARBINARY", "TINYBLOB", "MEDIUMBLOB", "BLOB", "LONGBLOB", "TINYTEXT", "MEDIUMTEXT",
		"TEXT", "LONGTEXT", "ENUM", "SET", "JSON", "DATE", "DATETIME", "TIMESTAMP", "TIME", "YEAR",
		"BIT", "SMALLINT", "TINYINT", "MEDIUMINT", "INT", "INTEGER", "BIGINT", "DECIMAL", "GEOMETRY",
		"POINT", "LINESTRING", "POLYGON", "MULTIPOINT", "MULTILINESTRING", "MULTIPOLYGON", "GEOMETRYCOLLECTION",
		"SIGNED", "UNSIGNED", "INTERVAL DAY_SECOND":
		return nil
	default:
		return errors.Errorf("unsupported type keyword %q", n.Keyword())
	}
}

func validateBareIdentifier(label string, ident lex.Ident) error {
	if !ident.Present() {
		return nil
	}
	if ident.Name == "" {
		return errors.Errorf("invalid type %s %q", label, ident.Name)
	}
	for i := 0; i < len(ident.Name); i++ {
		c := ident.Name[i]
		valid := c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
		if i > 0 {
			valid = valid || c >= '0' && c <= '9'
		}
		if !valid {
			return errors.Errorf("invalid type %s %q", label, ident.Name)
		}
	}
	return nil
}

func (n *TypeName) validateQualifiers() error {
	if err := validateBareIdentifier("charset", n.Charset); err != nil {
		return err
	}
	return validateBareIdentifier("collation", n.Collation)
}

func (n *TypeName) writeModifiers(ctx *format.RestoreCtx, count int, separator string) error {
	ctx.WritePlain("(")
	for i := range count {
		if i != 0 {
			ctx.WritePlain(separator)
		}
		if n.Modifiers[i] == nil {
			return errors.New("TypeName.Modifiers contains nil")
		}
		if err := n.Modifiers[i].Restore(ctx); err != nil {
			return errors.Annotate(err, "An error occurred while restore TypeName.Modifiers")
		}
	}
	ctx.WritePlain(")")
	return ctx.Err()
}

func (n *TypeName) Restore(ctx *format.RestoreCtx) error {
	if n == nil {
		return nilRestoreError("TypeName")
	}
	if err := n.validateKeyword(); err != nil {
		return err
	}
	if err := n.validateQualifiers(); err != nil {
		return err
	}
	keyword := n.restoreKeyword()
	if keyword == "" {
		return restoreQualifiedName(ctx, n.Name)
	}
	ctx.WriteKeyword(keyword)
	width, hasWidth := n.number(0)
	_, hasScale := n.number(1)
	count := 0
	switch n.Keyword() {
	case "BOOL", "BOOLEAN":
		ctx.WritePlain("(1)")
	case "UUID":
		ctx.WritePlain("(36)")
	case "INET6":
		ctx.WritePlain("(16)")
	case "ENUM", "SET":
		if err := n.writeModifiers(ctx, len(n.Modifiers), ","); err != nil {
			return err
		}
	case "TIMESTAMP", "DATETIME", "TIME":
		if hasWidth && width > 0 {
			count = 1
		}
	case "DOUBLE", "FLOAT", "DECIMAL":
		// An explicit zero scale must survive restore; only a missing
		// scale is dropped.
		if hasWidth && width > 0 && hasScale {
			count = 2
		}
	case "BIT", "SMALLINT", "TINYINT", "MEDIUMINT", "INT", "BIGINT", "CHAR", "VARCHAR",
		"BINARY", "VARBINARY", "TINYBLOB", "MEDIUMBLOB", "BLOB", "LONGBLOB",
		"TINYTEXT", "MEDIUMTEXT", "TEXT", "LONGTEXT", "YEAR":
		if hasWidth && width > 0 {
			count = 1
		}
	}
	if count != 0 {
		if err := n.writeModifiers(ctx, count, ","); err != nil {
			return err
		}
	}
	if n.Unsigned || n.Keyword() == "INET4" {
		ctx.WriteKeyword(" UNSIGNED")
	}
	if n.Zerofill {
		ctx.WriteKeyword(" ZEROFILL")
	}
	if n.Binary && !strings.EqualFold(n.Charset.Name, "binary") {
		ctx.WriteKeyword(" BINARY")
	}
	switch keyword {
	case "CHAR", "VARCHAR", "BINARY", "VARBINARY", "TINYTEXT", "TEXT", "MEDIUMTEXT", "LONGTEXT",
		"TINYBLOB", "BLOB", "MEDIUMBLOB", "LONGBLOB":
		charset := n.Charset.Name
		if n.Keyword() == "UUID" {
			charset = mysql.DefaultCharset
		}
		if charset != "" && !strings.EqualFold(charset, "binary") {
			ctx.WriteKeyword(" CHARACTER SET ")
			ctx.WritePlain(strings.ToLower(charset))
		}
		if n.Collation.Name != "" && !strings.EqualFold(n.Collation.Name, "binary") {
			ctx.WriteKeyword(" COLLATE ")
			ctx.WritePlain(strings.ToLower(n.Collation.Name))
		}
	}
	return ctx.Err()
}

func (n *TypeName) RestoreAsCastType(ctx *format.RestoreCtx) error {
	if err := n.validateKeyword(); err != nil {
		return err
	}
	if err := n.validateQualifiers(); err != nil {
		return err
	}
	keyword := n.Keyword()
	if width, ok := n.number(0); keyword == "FLOAT" && ok && width > 24 {
		keyword = "DOUBLE"
	}
	ctx.WriteKeyword(keyword)
	switch keyword {
	case "CHAR", "VARCHAR", "DATETIME", "TIME", "INTERVAL DAY_SECOND", "BINARY":
		if len(n.Modifiers) != 0 {
			if err := n.writeModifiers(ctx, 1, ","); err != nil {
				return err
			}
		}
	case "DECIMAL":
		width, hasWidth := n.number(0)
		_, hasScale := n.number(1)
		if !hasWidth {
			ctx.WritePlain("(11)")
		} else if width > 0 {
			count := 1
			// Match Restore: an explicit zero scale is preserved.
			if hasScale {
				count = 2
			}
			if err := n.writeModifiers(ctx, count, ", "); err != nil {
				return err
			}
		}
	}
	if keyword == "CHAR" || keyword == "VARCHAR" {
		if n.Binary {
			ctx.WriteKeyword(" BINARY")
		}
		if n.Charset.Name != "" && !strings.EqualFold(n.Charset.Name, "binary") &&
			!strings.EqualFold(n.Charset.Name, mysql.DefaultCharset) {
			ctx.WriteKeyword(" CHARACTER SET ")
			ctx.WritePlain(strings.ToLower(n.Charset.Name))
		}
	}
	return ctx.Err()
}

func (n *TypeName) FormatAsCastType(w io.Writer) {
	ctx := format.NewRestoreCtx(format.DefaultRestoreFlags, w)
	if err := n.RestoreAsCastType(ctx); err != nil {
		_, _ = fmt.Fprint(w, "<unsupported>")
	}
}

func (n *TypeName) Accept(v Visitor) (Node, bool) {
	if n == nil {
		return nil, false
	}
	entered, skip := v.Enter(n)
	outer, ok := entered.(*TypeName)
	if !ok || outer == nil {
		return entered, false
	}
	if skip {
		return v.Leave(outer)
	}
	for i, child := range outer.Modifiers {
		if !nodePresent(child) {
			// Restore rejects a missing modifier; Accept must fail instead
			// of silently skipping the entry.
			return outer, false
		}
		replacement, ok := child.Accept(v)
		if !ok {
			return outer, false
		}
		expr, ok := replacement.(ExprNode)
		if !ok || !nodePresent(expr) {
			return outer, false
		}
		outer.Modifiers[i] = expr
	}
	return v.Leave(outer)
}

var _ pkast.TypeNode = (*TypeName)(nil)
