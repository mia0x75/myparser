package myparser

import (
	io0 "io"

	"os"
	"path"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"
)

func TestKeywordConsistent(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	parserFilename := path.Join(path.Dir(filename), "parser.y")
	parserFile, err := os.Open(parserFilename)
	if err != nil {
		t.Fatalf("expected nil, got %#v", err)
	}
	data, err := io0.ReadAll(parserFile)
	if err != nil {
		t.Fatalf("expected nil, got %#v", err)
	}
	content := string(data)

	reservedKeywordStartMarker := "\t/* The following tokens belong to ReservedKeyword. */"
	unreservedKeywordStartMarker := "\t/* The following tokens belong to UnReservedKeyword. */"
	notKeywordTokenStartMarker := "\t/* The following tokens belong to NotKeywordToken. */"
	identTokenEndMarker := "%token\t<item>"

	reservedKeywords := extractTokenNames(content, reservedKeywordStartMarker, unreservedKeywordStartMarker)

	unreservedKeywords := extractTokenNames(content, unreservedKeywordStartMarker, notKeywordTokenStartMarker)
	notKeywordTokens := extractTokenNames(content, notKeywordTokenStartMarker, identTokenEndMarker)
	for k, v := range aliases {
		if k == v {
			t.Fatalf("expected true")
		}
		if tokenMap[k] != tokenMap[v] {
			t.Fatalf("got %#v, want %#v", tokenMap[k], tokenMap[v])
		}
	}
	keywordCount := len(reservedKeywords) + len(unreservedKeywords) + len(notKeywordTokens)
	if len(tokenMap)-len(aliases) != keywordCount-len(windowFuncTokenMap) {
		t.Fatalf("got %#v, want %#v", len(tokenMap)-len(aliases), keywordCount-len(windowFuncTokenMap))
	}

	unreservedCollectionDef := extractIdentifiersFromCollectionDef(content, "\nUnReservedKeyword:")
	if !reflect.DeepEqual(unreservedKeywords, unreservedCollectionDef) {
		t.Fatalf("got %#v, want %#v", unreservedKeywords, unreservedCollectionDef)
	}

	notKeywordTokensCollectionDef := extractIdentifiersFromCollectionDef(content, "\nNotKeywordToken:")
	if !reflect.DeepEqual(notKeywordTokens, notKeywordTokensCollectionDef) {
		t.Fatalf("got %#v, want %#v", notKeywordTokens, notKeywordTokensCollectionDef)
	}
}

func extractMiddle(str, startMarker, endMarker string) string {
	startIdx := strings.Index(str, startMarker)
	if startIdx == -1 {
		return ""
	}
	str = str[startIdx+len(startMarker):]
	before0, _, ok := strings.Cut(str, endMarker)
	if !ok {
		return ""
	}
	return before0
}

func extractTokenNames(content, startMarker, endMarker string) []string {
	section := extractMiddle(content, startMarker, endMarker)
	lines := strings.Split(section, "\n")
	var names []string
	for _, line := range lines {
		t := strings.TrimSpace(line)
		if t == "" || strings.HasPrefix(t, "//") {
			continue
		}
		if strings.HasPrefix(t, "/*") {
			if i := strings.Index(t, "*/"); i >= 0 {
				t = strings.TrimSpace(t[i+2:])
			}
		}
		if !strings.Contains(t, "\"") {
			continue
		}
		fields := strings.Fields(t)
		if len(fields) == 0 {
			continue
		}
		names = append(names, fields[0])
	}
	sort.Strings(names)
	return names
}

func extractIdentifiersFromCollectionDef(content, startMarker string) []string {
	keywordSection := extractMiddle(content, startMarker, "\n\n")
	parts := strings.Split(keywordSection, "|")
	var names []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if i := strings.Index(part, "%prec"); i >= 0 {
			part = strings.TrimSpace(part[:i])
		}
		fields := strings.Fields(part)
		if len(fields) == 0 {
			continue
		}
		names = append(names, fields[0])
	}
	sort.Strings(names)
	return names
}
