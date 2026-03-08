package md2html

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestParseMarkdownToAST(t *testing.T) {
	fixtures := []string{
		"paragraphs",
		"headings",
		"inline_styles",
		"lists",
		"nested_lists",
		"tables",
		"code_blocks",
		"thematic_break",
		"links",
		"images",
	}

	for _, name := range fixtures {
		t.Run(name, func(t *testing.T) {
			dir := filepath.Join("testdata", "elements", name)
			inputPath := filepath.Join(dir, "input.md")
			wantPath := filepath.Join(dir, "ast.json")

			input, err := os.ReadFile(inputPath)
			if err != nil {
				t.Fatalf("read input: %v", err)
			}

			got, err := ParseMarkdownToAST(string(input))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}

			gotJSON, err := json.MarshalIndent(got, "", "  ")
			if err != nil {
				t.Fatalf("marshal got: %v", err)
			}

			if _, err := os.Stat(wantPath); os.IsNotExist(err) {
				// First run: write golden file
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatalf("mkdir: %v", err)
				}
				if err := os.WriteFile(wantPath, gotJSON, 0644); err != nil {
					t.Fatalf("write golden: %v", err)
				}
				t.Logf("wrote golden file %s", wantPath)
				return
			}

			wantJSON, err := os.ReadFile(wantPath)
			if err != nil {
				t.Fatalf("read golden: %v", err)
			}

			if string(gotJSON) != string(wantJSON) {
				t.Errorf("AST mismatch for %s.\nGot:\n%s\nWant:\n%s", name, gotJSON, wantJSON)
			}
		})
	}
}

