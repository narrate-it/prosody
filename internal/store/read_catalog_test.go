package store_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"reflect"
	"strconv"
	"testing"

	"github.com/narrate-it/prosody/internal/store"
)

// These hashes freeze the seven original literal callsites at main 9b5a922.
// SQL whitespace and all publication-count subqueries are part of the contract.
var readHashes = map[string]string{
	"feeds":         "b7c2e5f1b89accc26294ee5810415f967d9ed23804f0206c6a30cf7a516fce00",
	"clock":         "92da62dc3b3f7b341c4a327909755f3e26e99a747f830cef79e3a582a80b7556",
	"readable-feed": "7c5b6875f75832209bd0ae01c804cb14f7b884152bbb842f397e8b01f92d4108",
	"activity":      "6abe76f4608c74eef17898aaef4c08439dd65e3728aeb1c2e56127c2e9ebe548",
	"ready-list":    "90fa33c94380af96600f8451ce3b2b0fbe80a265a10030ef699ea58aa2932b9d",
	"ready-detail":  "11ced9a211a89c477221fc895871bebdb9754fee9bde95bc4bddc0150b13b36e",
}

func TestReadCatalogExactOriginalSQLHashes(t *testing.T) {
	t.Parallel()
	catalog := store.ConsumptionReadCatalog()
	if len(catalog) != len(readHashes) {
		t.Fatalf("statements=%d", len(catalog))
	}
	seen := map[string]bool{}
	for _, s := range catalog {
		t.Run(s.ID, func(t *testing.T) {
			expected, exists := readHashes[s.ID]
			if !exists || seen[s.ID] {
				t.Fatalf("unknown or duplicate ID %q", s.ID)
			}
			seen[s.ID] = true
			actual := sha256.Sum256([]byte(s.SQL))
			if actual != s.SHA256 || hex.EncodeToString(actual[:]) != expected {
				t.Fatal("exact original SQL bytes changed")
			}
			if s.Phase != "human-read" || len(s.NativeIdentitySlots) != 0 || s.NativeIdentityNote == "" {
				t.Fatal("missing local/native distinction")
			}
			if s.ID == "feeds" && (len(s.Columns) != 3 || s.Columns[2].Name != "title" || s.Columns[2].SQLType != "bytea" || s.Columns[2].MaxBytes != 4096) {
				t.Fatal("feed title SQL bytes must not be mislabeled as presentation text")
			}
			if s.ID == "clock" {
				if s.WorkspaceParameter != 0 || len(s.Arguments) != 0 || s.SQL != "SELECT pg_catalog.clock_timestamp()" {
					t.Fatal("clock exception broadened")
				}
			} else if s.WorkspaceParameter != 1 {
				t.Fatal("workspace binding changed")
			}
			for i, a := range s.Arguments {
				if a.Position != i+1 {
					t.Fatal("argument ordinal changed")
				}
			}
		})
	}
}

func TestReadCatalogCallsitesAndArguments(t *testing.T) {
	t.Parallel()
	expected := map[string][]string{
		"ReadFeeds":                {"readFeedsSQL", "workspace", "after", "limit + 1"},
		"ReadReadyBriefings/clock": {"readClockSQL"},
		"ReadReadyBriefings/list":  {"readReadyBriefingsSQL", "workspace", "feed", "c.Cutoff", "cursor != \"\"", "c.PublishedAt", "c.ItemID", "c.RenditionID", "limit + 1"},
		"ReadReadyBriefing":        {"readReadyBriefingSQL", "workspace", "feed", "item"},
		"lockReadableFeed":         {"readableFeedSQL", "workspace", "feed"},
		"ReadActivity/clock":       {"readClockSQL"},
		"ReadActivity/list":        {"readActivitySQL", "workspace", "feed", "c.Cutoff", "cursor != \"\"", "c.AcceptedAt", "c.ItemID", "limit + 1"},
	}
	seen := map[string]bool{}
	fset := token.NewFileSet()
	for _, path := range []string{"reads.go", "activity.go"} {
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			// Existing FromSQLRead only forwards callback SQL; it owns no literal.
			if fn.Name.Name == "FromSQLRead" {
				continue
			}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				selector, ok := call.Fun.(*ast.SelectorExpr)
				if !ok || (selector.Sel.Name != "QueryContext" && selector.Sel.Name != "QueryRowContext") {
					return true
				}
				var got []string
				for _, arg := range call.Args[1:] {
					var b bytes.Buffer
					if err := format.Node(&b, fset, arg); err != nil {
						t.Fatal(err)
					}
					got = append(got, b.String())
				}
				key := fn.Name.Name
				if key == "ReadReadyBriefings" || key == "ReadActivity" {
					if got[0] == "readClockSQL" {
						key += "/clock"
					} else {
						key += "/list"
					}
				}
				want, exists := expected[key]
				if !exists || seen[key] || !reflect.DeepEqual(got, want) {
					t.Errorf("%s args=%v want=%v", key, got, want)
				}
				seen[key] = true
				return true
			})
		}
	}
	if len(seen) != 7 {
		t.Fatalf("query callsites=%d", len(seen))
	}
	// Check the constants themselves are exact literal bytes, not dynamically
	// generated SQL or concatenated fragments that evade the source inventory.
	file, err := parser.ParseFile(fset, "read_catalog.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	literals := map[string]string{}
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST {
			continue
		}
		for _, spec := range gen.Specs {
			v := spec.(*ast.ValueSpec)
			if len(v.Values) != 1 {
				t.Fatal("nonliteral SQL declaration")
			}
			literal, ok := v.Values[0].(*ast.BasicLit)
			if !ok {
				t.Fatal("nonliteral SQL")
			}
			s, err := strconv.Unquote(literal.Value)
			if err != nil {
				t.Fatal(err)
			}
			literals[v.Names[0].Name] = s
		}
	}
	if len(literals) != 6 {
		t.Fatalf("literal constants=%d", len(literals))
	}
	for _, want := range expected {
		if _, exists := literals[want[0]]; !exists {
			t.Fatal("uncataloged SQL reference")
		}
	}
	for _, s := range store.ConsumptionReadCatalog() {
		found := false
		for _, sql := range literals {
			if sql == s.SQL {
				found = true
			}
		}
		if !found {
			t.Fatal("catalog differs from literal")
		}
	}
}

func TestReadCatalogReturnsIndependentMetadata(t *testing.T) {
	t.Parallel()
	expected := store.ConsumptionReadCatalog()
	mutated := store.ConsumptionReadCatalog()
	mutated[0].ID = "changed"
	mutated[0].Arguments[0].Role = "authority"
	mutated[0].Columns[0].Name = "secret"
	mutated[0].SHA256[0]++
	mutated[0].NativeIdentitySlots = append(mutated[0].NativeIdentitySlots, "forged")
	mutated[4].Columns[0].Name = "shared column mutation"
	if !reflect.DeepEqual(expected, store.ConsumptionReadCatalog()) {
		t.Fatal("catalog metadata aliases canonical registry")
	}
	if expected[5].Columns[0].Name != "item_id" {
		t.Fatal("result column slices alias")
	}
	want := store.ConsumptionReadBudgets()
	got := store.ConsumptionReadBudgets()
	got[0].Statements[0] = "foreign"
	got[0].MaxCalls = 100
	if !reflect.DeepEqual(want, store.ConsumptionReadBudgets()) {
		t.Fatal("route metadata aliases canonical registry")
	}
}

func TestReadCatalogSeparatesSourceAndInitialLimits(t *testing.T) {
	t.Parallel()
	roles := map[string][]string{
		"feeds":         {"workspace", "cursor-location", "page-limit"},
		"clock":         {},
		"readable-feed": {"workspace", "resource-feed"},
		"activity":      {"workspace", "resource-feed", "cursor-location", "cursor-location", "cursor-location", "cursor-location", "page-limit"},
		"ready-list":    {"workspace", "resource-feed", "cursor-location", "cursor-location", "cursor-location", "cursor-location", "cursor-location", "page-limit"},
		"ready-detail":  {"workspace", "resource-feed", "resource-item"},
	}
	for _, s := range store.ConsumptionReadCatalog() {
		var got []string
		for _, a := range s.Arguments {
			got = append(got, a.Role)
			if a.Role == "page-limit" && (a.SourceMax != 51 || a.InitialMax != 11) {
				t.Fatal("lookahead limits conflated")
			}
		}
		if len(got) != len(roles[s.ID]) {
			t.Fatal("slot count changed")
		}
		for i := range got {
			if got[i] != roles[s.ID][i] {
				t.Fatal("cursor or resource role changed")
			}
		}
		wantRows, wantInitial := 1, 1
		if s.ID == "feeds" || s.ID == "activity" || s.ID == "ready-list" {
			wantRows, wantInitial = 51, 11
		}
		if s.SourceMaxRows != wantRows || s.InitialMaxRows != wantInitial {
			t.Fatal("source/initial row ceilings conflated")
		}
		if s.ID == "ready-list" || s.ID == "ready-detail" {
			if s.MaxVariableBytesPerRow != 65536+256000+262144 {
				t.Fatal("payload byte bounds incomplete")
			}
		}
	}
	calls := map[string]int{"feeds": 1, "activity": 3, "ready-list": 3, "ready-detail": 2}
	for _, b := range store.ConsumptionReadBudgets() {
		if b.MaxCalls != calls[b.ID] || len(b.Statements) != b.MaxCalls || b.MaxCallsPerStatement != 1 {
			t.Fatal("route query budget changed")
		}
		seen := map[string]bool{}
		for _, id := range b.Statements {
			if _, exists := readHashes[id]; !exists || seen[id] {
				t.Fatal("unknown/duplicate route statement")
			}
			seen[id] = true
		}
		if b.ID != "ready-detail" && (b.SourcePageCeiling != 50 || b.InitialPageCeiling != 10) {
			t.Fatal("page caps conflated")
		}
		if b.InitialMaxRawPayloadBytes != 8<<20 || b.InitialMaxEncodedJSONBytes != 1<<20 {
			t.Fatal("proposed byte budgets changed")
		}
	}
	// Existing code's general source ceiling remains untouched by registration
	// preparation; this structural assertion is not native admission proof.
	source, err := os.ReadFile("reads.go")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(source, []byte("const maxReadPage = 50")) {
		t.Fatal("source page behavior changed")
	}
}
