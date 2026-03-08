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

func TestASTToIR_MixedTopLevelListTypesStaySeparated(t *testing.T) {
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

	if len(lists) != 4 {
		t.Fatalf("expected 4 top-level lists (ol, ul, ol, ul), got %d", len(lists))
	}

	wantOrdered := []bool{true, false, true, false}
	for i := range wantOrdered {
		if lists[i].Ordered != wantOrdered[i] {
			t.Fatalf("list %d ordered mismatch: got %v want %v", i, lists[i].Ordered, wantOrdered[i])
		}
		if len(lists[i].Items) == 0 {
			t.Fatalf("list %d has no items", i)
		}
	}

	firstTexts := []string{
		segmentsToPlainText(lists[0].Items[0].Segments),
		segmentsToPlainText(lists[1].Items[0].Segments),
		segmentsToPlainText(lists[2].Items[0].Segments),
		segmentsToPlainText(lists[3].Items[0].Segments),
	}
	wantTexts := []string{"Ingest", "Input EPUB", "User model", "Keep a per-user profile"}
	for i := range wantTexts {
		if firstTexts[i] != wantTexts[i] {
			t.Fatalf("list %d first item mismatch: got %q want %q", i, firstTexts[i], wantTexts[i])
		}
	}
}
