package md2html

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestASTToIR(t *testing.T) {
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
		"box_drawing",
	}

	for _, name := range fixtures {
		t.Run(name, func(t *testing.T) {
			dir := filepath.Join("testdata", "elements", name)
			astPath := filepath.Join(dir, "ast.json")
			wantPath := filepath.Join(dir, "ir.json")

			astJSON, err := os.ReadFile(astPath)
			if err != nil {
				t.Fatalf("read ast: %v", err)
			}

			var astDoc Document
			if err := json.Unmarshal(astJSON, &astDoc); err != nil {
				t.Fatalf("unmarshal ast: %v", err)
			}

			got := ASTToIR(astDoc)

			gotJSON, err := json.MarshalIndent(got, "", "  ")
			if err != nil {
				t.Fatalf("marshal got: %v", err)
			}

			if _, err := os.Stat(wantPath); os.IsNotExist(err) {
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
				t.Errorf("IR mismatch for %s.\nGot:\n%s\nWant:\n%s", name, gotJSON, wantJSON)
			}
		})
	}
}

func TestASTToIR_ChatBlocks(t *testing.T) {
	dir := filepath.Join("testdata", "elements", "chat_blocks")
	inputPath := filepath.Join(dir, "input.md")
	wantPath := filepath.Join(dir, "ir.json")

	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	input := "```prompt\nHello **world**\n```\n\n---\n\n```agent\nResponse with `code`.\n```\n"
	if err := os.WriteFile(inputPath, []byte(input), 0644); err != nil {
		t.Fatalf("write input: %v", err)
	}

	astDoc, err := ParseMarkdownToAST(input)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	got := ASTToIR(astDoc)

	gotJSON, err := json.MarshalIndent(got, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	if _, err := os.Stat(wantPath); os.IsNotExist(err) {
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
		t.Errorf("IR mismatch for chat_blocks.\nGot:\n%s\nWant:\n%s", gotJSON, wantJSON)
	}
}

func TestASTToIR_MixedTopLevelListBecomesOrderedWithBulletChildren(t *testing.T) {
	input := `1. Ingest

- Input EPUB
- Parse chapters/paragraphs/sentences

2. User model

- Keep a per-user profile`

	astDoc, err := ParseMarkdownToAST(input)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	ir := ASTToIR(astDoc)

	lists := make([]*IRList, 0)
	for _, b := range ir.Blocks {
		if b.List != nil {
			lists = append(lists, b.List)
		}
	}

	if len(lists) != 1 {
		t.Fatalf("expected 1 top-level ordered list, got %d", len(lists))
	}
	if !lists[0].Ordered {
		t.Fatalf("expected top-level list to be ordered")
	}
	if len(lists[0].Items) != 2 {
		t.Fatalf("expected 2 ordered items, got %d", len(lists[0].Items))
	}

	orderedTexts := []string{
		segmentsToPlainText(lists[0].Items[0].Segments),
		segmentsToPlainText(lists[0].Items[1].Segments),
	}
	if orderedTexts[0] != "Ingest" || orderedTexts[1] != "User model" {
		t.Fatalf("unexpected ordered item texts: %+v", orderedTexts)
	}

	if lists[0].Items[0].Children == nil || lists[0].Items[1].Children == nil {
		t.Fatalf("expected bullet children for both ordered items")
	}
	if lists[0].Items[0].Children.Ordered || lists[0].Items[1].Children.Ordered {
		t.Fatalf("expected child lists to be unordered")
	}
	if len(lists[0].Items[0].Children.Items) != 2 || len(lists[0].Items[1].Children.Items) != 1 {
		t.Fatalf("unexpected child list sizes: first=%d second=%d",
			len(lists[0].Items[0].Children.Items), len(lists[0].Items[1].Children.Items))
	}
}
