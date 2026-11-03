package store

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"regexp"
	"strconv"
	"testing"
)

// This checks the review artifact, not SQL execution, grants or native admission.
func TestCheckpointCatalogMatchesSource(t *testing.T) {
	raw, err := os.ReadFile("../../docs/implementation/CHECKPOINT-SQL.json")
	if err != nil {
		t.Fatal(err)
	}
	var catalog struct {
		BindingEncoding struct {
			Version int `json:"version"`
		} `json:"binding_encoding"`
		MigrationDigest string `json:"migration_sha256"`
		Statements      []struct {
			ID        string `json:"id"`
			SQL       string `json:"sql"`
			Digest    string `json:"sha256"`
			Slots     []int  `json:"ordinal_slots"`
			Arguments []struct {
				Slot int `json:"slot"`
			} `json:"arguments"`
		} `json:"statements"`
	}
	if json.Unmarshal(raw, &catalog) != nil || catalog.BindingEncoding.Version != CheckpointBindingEncodingVersion {
		t.Fatal("catalog encoding mismatch")
	}
	migration := sha256.Sum256([]byte(CheckpointMigration))
	if catalog.MigrationDigest != hex.EncodeToString(migration[:]) {
		t.Fatal("migration digest mismatch")
	}
	expected := map[string]string{"CheckpointSourceSQL": CheckpointSourceSQL, "CheckpointInsertSQL": CheckpointInsertSQL, "CheckpointReadSQL": CheckpointReadSQL}
	slots := regexp.MustCompile(`\$([0-9]+)`)
	for _, statement := range catalog.Statements {
		want, ok := expected[statement.ID]
		if !ok || want != statement.SQL {
			t.Fatal("missing, duplicate or changed statement")
		}
		delete(expected, statement.ID)
		digest := sha256.Sum256([]byte(want))
		if statement.Digest != hex.EncodeToString(digest[:]) {
			t.Fatal("statement digest mismatch")
		}
		actual := map[int]bool{}
		for _, match := range slots.FindAllStringSubmatch(want, -1) {
			n, err := strconv.Atoi(match[1])
			if err != nil {
				t.Fatal(err)
			}
			actual[n] = true
		}
		if len(actual) != len(statement.Slots) || len(actual) != len(statement.Arguments) {
			t.Fatal("argument inventory length mismatch")
		}
		for i, n := range statement.Slots {
			if n != i+1 || !actual[n] || statement.Arguments[i].Slot != n {
				t.Fatal("argument ordinal mismatch")
			}
		}
	}
	if len(expected) != 0 {
		t.Fatal("missing catalog statements")
	}
}
