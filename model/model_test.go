package model

import (
	"encoding/json"
	"math"
	"reflect"
	"testing"

	"github.com/mia0x75/myparser/mysql"
	"github.com/mia0x75/myparser/types"
)

func TestT(t *testing.T) {
	abc := NewCIStr("aBC")
	if abc.O != "aBC" {
		t.Fatalf("got %#v, want %#v", abc.O, "aBC")
	}
	if abc.L != "abc" {
		t.Fatalf("got %#v, want %#v", abc.L, "abc")
	}
	if abc.String() != "aBC" {
		t.Fatalf("got %#v, want %#v", abc.String(), "aBC")
	}
}

func TestModelBasic(t *testing.T) {
	column := &ColumnInfo{
		ID:           1,
		Name:         NewCIStr("c"),
		Offset:       0,
		DefaultValue: 0,
		FieldType:    *types.NewFieldType(0),
	}
	column.Flag |= mysql.PriKeyFlag

	index := &IndexInfo{
		Name:  NewCIStr("key"),
		Table: NewCIStr("t"),
		Columns: []*IndexColumn{
			{
				Name:   NewCIStr("c"),
				Offset: 0,
				Length: 10,
			}},
		Unique:  true,
		Primary: true,
	}

	fk := &FKInfo{
		RefCols: []CIStr{NewCIStr("a")},
		Cols:    []CIStr{NewCIStr("a")},
	}

	table := &TableInfo{
		ID:          1,
		Name:        NewCIStr("t"),
		Charset:     "utf8",
		Collate:     "utf8_bin",
		Columns:     []*ColumnInfo{column},
		Indices:     []*IndexInfo{index},
		ForeignKeys: []*FKInfo{fk},
		PKIsHandle:  true,
	}

	dbInfo := &DBInfo{
		ID:      1,
		Name:    NewCIStr("test"),
		Charset: "utf8",
		Collate: "utf8_bin",
		Tables:  []*TableInfo{table},
	}

	n := dbInfo.Clone()
	if !reflect.DeepEqual(n, dbInfo) {
		t.Fatalf("got %#v, want %#v", n, dbInfo)
	}

	pkName := table.GetPkName()
	if !reflect.DeepEqual(pkName, NewCIStr("c")) {
		t.Fatalf("got %#v, want %#v", pkName, NewCIStr("c"))
	}
	newColumn := table.GetPkColInfo()
	if !reflect.DeepEqual(newColumn, column) {
		t.Fatalf("got %#v, want %#v", newColumn, column)
	}
	inIdx := table.ColumnIsInIndex(column)
	if inIdx != true {
		t.Fatalf("got %#v, want %#v", inIdx, true)
	}
	tp := IndexTypeBtree
	if tp.String() != "BTREE" {
		t.Fatalf("got %#v, want %#v", tp.String(), "BTREE")
	}
	tp = IndexTypeHash
	if tp.String() != "HASH" {
		t.Fatalf("got %#v, want %#v", tp.String(), "HASH")
	}
	tp = 1e5
	if tp.String() != "" {
		t.Fatalf("got %#v, want %#v", tp.String(), "")
	}
	has := index.HasPrefixIndex()
	if has != true {
		t.Fatalf("got %#v, want %#v", has, true)
	}
	updateTime := table.GetUpdateTime()
	if !reflect.DeepEqual(updateTime, TSConvert2Time(table.UpdateTS)) {
		t.Fatalf("got %#v, want %#v", updateTime, TSConvert2Time(table.UpdateTS))
	}

	// Corner cases
	column.Flag ^= mysql.PriKeyFlag
	pkName = table.GetPkName()
	if !reflect.DeepEqual(pkName, NewCIStr("")) {
		t.Fatalf("got %#v, want %#v", pkName, NewCIStr(""))
	}
	newColumn = table.GetPkColInfo()
	if newColumn != nil {
		t.Fatalf("got %#v, want nil", newColumn)
	}
	anCol := &ColumnInfo{
		Name: NewCIStr("d"),
	}
	exIdx := table.ColumnIsInIndex(anCol)
	if exIdx != false {
		t.Fatalf("got %#v, want %#v", exIdx, false)
	}
	anIndex := &IndexInfo{
		Columns: []*IndexColumn{},
	}
	no := anIndex.HasPrefixIndex()
	if no != false {
		t.Fatalf("got %#v, want %#v", no, false)
	}
}

func TestString(t *testing.T) {
	acts := []struct {
		act    ActionType
		result string
	}{
		{ActionNone, "none"},
		{ActionAddForeignKey, "add foreign key"},
		{ActionDropForeignKey, "drop foreign key"},
		{ActionTruncateTable, "truncate table"},
		{ActionModifyColumn, "modify column"},
		{ActionRenameTable, "rename table"},
		{ActionSetDefaultValue, "set default value"},
		{ActionCreateSchema, "create schema"},
		{ActionDropSchema, "drop schema"},
		{ActionCreateTable, "create table"},
		{ActionDropTable, "drop table"},
		{ActionAddIndex, "add index"},
		{ActionDropIndex, "drop index"},
		{ActionAddColumn, "add column"},
		{ActionDropColumn, "drop column"},
		{ActionModifySchemaCharsetAndCollate, "modify schema charset and collate"},
	}

	for _, v := range acts {
		str := v.act.String()
		if str != v.result {
			t.Fatalf("got %#v, want %#v", str, v.result)
		}
	}
}

func TestModelDefaultAndGenerated(t *testing.T) {
	// IsGenerated
	col := &ColumnInfo{Name: NewCIStr("c"), GeneratedExprString: "a + b"}
	if !col.IsGenerated() {
		t.Fatal("IsGenerated: want true")
	}
	col.GeneratedExprString = ""
	if col.IsGenerated() {
		t.Fatal("IsGenerated: want false")
	}

	// SetDefaultValue / GetDefaultValue
	col.Tp = mysql.TypeBit
	if err := col.SetDefaultValue("01"); err != nil {
		t.Fatalf("SetDefaultValue: %v", err)
	}
	if got := col.GetDefaultValue(); got != "01" {
		t.Fatalf("GetDefaultValue: got %#v, want %q", got, "01")
	}
	if err := col.SetDefaultValue(1); err == nil {
		t.Fatal("SetDefaultValue: want error for int bit value")
	}
	col.Tp = 0
	if err := col.SetDefaultValue(42); err != nil {
		t.Fatalf("SetDefaultValue: %v", err)
	}
	if got := col.GetDefaultValue(); got != 42 {
		t.Fatalf("GetDefaultValue: got %#v, want 42", got)
	}

	// FindColumnInfo
	cols := []*ColumnInfo{{Name: NewCIStr("A")}, {Name: NewCIStr("b")}}
	if got := FindColumnInfo(cols, "B"); got == nil || got.Name.O != "b" {
		t.Fatalf("FindColumnInfo: got %#v", got)
	}
	if got := FindColumnInfo(cols, "x"); got != nil {
		t.Fatalf("FindColumnInfo: got %#v, want nil", got)
	}
}

func TestModelTableInfo0Percent(t *testing.T) {
	table := &TableInfo{
		Name:    NewCIStr("t"),
		Columns: []*ColumnInfo{{Name: NewCIStr("a"), Flag: mysql.PriKeyFlag}},
	}

	// GetAutoIncrementColInfo / IsAutoIncColUnsigned
	autoCol := &ColumnInfo{Name: NewCIStr("id"), Flag: mysql.AutoIncrementFlag}
	table.Columns = append(table.Columns, autoCol)
	if got := table.GetAutoIncrementColInfo(); got != autoCol {
		t.Fatalf("GetAutoIncrementColInfo: got %#v", got)
	}
	if table.IsAutoIncColUnsigned() {
		t.Fatal("IsAutoIncColUnsigned: want false for signed auto increment column")
	}
	autoCol.Flag |= mysql.UnsignedFlag
	if !table.IsAutoIncColUnsigned() {
		t.Fatal("IsAutoIncColUnsigned: want true for unsigned auto increment column")
	}
	autoCol.Flag &= ^mysql.UnsignedFlag

	// IsView
	view := &TableInfo{Name: NewCIStr("v")}
	view.View = &ViewInfo{SelectStmt: "SELECT 1"}
	if !view.IsView() {
		t.Fatal("IsView: want true")
	}
	if table.IsView() {
		t.Fatal("IsView: want false")
	}

	// GetDBID
	db := &DBInfo{ID: 7, Name: NewCIStr("db"), Tables: []*TableInfo{table}}
	if got := table.GetDBID(db.ID); got != 7 {
		t.Fatalf("GetDBID: got %d, want 7", got)
	}
}

func TestModelHistoryInfo(t *testing.T) {
	h := &HistoryInfo{}
	db := &DBInfo{ID: 1, Name: NewCIStr("db")}
	tbl := &TableInfo{ID: 2, Name: NewCIStr("t")}
	h.AddDBInfo(1, db)
	if h.SchemaVersion != 1 || h.DBInfo != db {
		t.Fatal("AddDBInfo")
	}
	h.AddTableInfo(2, tbl)
	if h.SchemaVersion != 2 || h.TableInfo != tbl {
		t.Fatal("AddTableInfo")
	}
	h.Clean()
	if h.SchemaVersion != 0 || h.DBInfo != nil || h.TableInfo != nil {
		t.Fatal("Clean")
	}
	if meta := NewDDLReorgMeta(); meta.EndHandle != math.MaxInt64 {
		t.Fatalf("NewDDLReorgMeta: EndHandle got %d, want %d", meta.EndHandle, int64(math.MaxInt64))
	}
}

func TestUnmarshalCIStr(t *testing.T) {
	var ci CIStr

	// Test unmarshal CIStr from a single string.
	str := "aaBB"
	buf, err := json.Marshal(str)
	if err != nil {
		t.Fatal(err)
	}
	if err := ci.UnmarshalJSON(buf); err != nil {
		t.Fatal(err)
	}
	if ci.O != str {
		t.Fatalf("got %#v, want %#v", ci.O, str)
	}
	if ci.L != "aabb" {
		t.Fatalf("got %#v, want %#v", ci.L, "aabb")
	}

	buf, err = json.Marshal(ci)
	if err != nil {
		t.Fatal(err)
	}
	if string(buf) != `{"O":"aaBB","L":"aabb"}` {
		t.Fatalf("got %#v, want %#v", string(buf), `{"O":"aaBB","L":"aabb"}`)
	}
	if err := ci.UnmarshalJSON(buf); err != nil {
		t.Fatal(err)
	}
	if ci.O != str {
		t.Fatalf("got %#v, want %#v", ci.O, str)
	}
	if ci.L != "aabb" {
		t.Fatalf("got %#v, want %#v", ci.L, "aabb")
	}
}
