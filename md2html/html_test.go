package md2html

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestIRToHTML(t *testing.T) {
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
		"chat_blocks",
	}

	for _, name := range fixtures {
		t.Run(name, func(t *testing.T) {
			dir := filepath.Join("testdata", "elements", name)
			irPath := filepath.Join(dir, "ir.json")
			wantPath := filepath.Join(dir, "output.html")

			irJSON, err := os.ReadFile(irPath)
			if err != nil {
				t.Fatalf("read ir: %v", err)
			}

			var irDoc IRDocument
			if err := json.Unmarshal(irJSON, &irDoc); err != nil {
				t.Fatalf("unmarshal ir: %v", err)
			}

			got := IRToHTML(irDoc)

			if _, err := os.Stat(wantPath); os.IsNotExist(err) {
				if err := os.WriteFile(wantPath, []byte(got), 0644); err != nil {
					t.Fatalf("write golden: %v", err)
				}
				t.Logf("wrote golden file %s", wantPath)
				return
			}

			want, err := os.ReadFile(wantPath)
			if err != nil {
				t.Fatalf("read golden: %v", err)
			}

			if got != string(want) {
				t.Errorf("HTML mismatch for %s.\nGot:\n%s\nWant:\n%s", name, got, want)
			}
		})
	}
}

// TestFullPipeline runs the complete md -> AST -> IR -> HTML pipeline.
func TestFullPipeline(t *testing.T) {
	input := `# Title

Some **bold** text.

---

` + "```prompt\nAsk a question\n```" + `

` + "```agent\nHere is the **answer**.\n```" + `

| A | B |
|---|---|
| 1 | 2 |
`

	ast, err := ParseMarkdownToAST(input)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	ir := ASTToIR(ast)
	html := IRToHTML(ir)

	// Sanity checks on the output.
	checks := []string{
		"<h1 id=\"title\"><a href=\"#title\">Title</a></h1>",
		"<strong>bold</strong>",
		"<hr>",
		"chat-prompt",
		"chat-agent",
		"<strong>answer</strong>",
		"<table>",
		"<th>A</th>",
		"<td>1</td>",
	}
	for _, check := range checks {
		if !contains(html, check) {
			t.Errorf("expected HTML to contain %q, got:\n%s", check, html)
		}
	}
}

func TestSlugify(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"Hello World", "hello-world"},
		{"Demo 2 - Google Docs (2)", "demo-2-google-docs-2"},
		{"What's new in v3.0?", "whats-new-in-v30"},
		{"---leading & trailing---", "leading-trailing"},
		{"ALL  CAPS   SPACES", "all-caps-spaces"},
		{"C++ Templates", "c-templates"},
		{"price: $100/unit", "price-100unit"},
		{"", ""},
		{"---", ""},
		{"a_b_c", "a-b-c"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got := slugify(tt.in)
			if got != tt.want {
				t.Errorf("slugify(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestUniqueSlug(t *testing.T) {
	slugs := make(map[string]int)
	if got := uniqueSlug("intro", slugs); got != "intro" {
		t.Errorf("first occurrence: got %q, want %q", got, "intro")
	}
	if got := uniqueSlug("intro", slugs); got != "intro-2" {
		t.Errorf("second occurrence: got %q, want %q", got, "intro-2")
	}
	if got := uniqueSlug("intro", slugs); got != "intro-3" {
		t.Errorf("third occurrence: got %q, want %q", got, "intro-3")
	}
	if got := uniqueSlug("other", slugs); got != "other" {
		t.Errorf("different slug: got %q, want %q", got, "other")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsSubstring(s, substr)
}

func containsSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
