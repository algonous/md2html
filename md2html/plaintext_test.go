package md2html

import "testing"

func TestMarkdownToPlainText(t *testing.T) {
	input := `# Status

So it's **not CPU** and ` + "`still bootstrapping`" + `.

Visit [Google](https://www.google.com) and https://example.com.

1. First
   - Child

` + "```go\nfmt.Println(\"ok\")\n```" + `

| A | B |
|---|---|
| 1 | 2 |

![Alt text](https://example.com/image.png)

` + "```prompt\nHello **world**.\n```" + `
`

	got, err := MarkdownToPlainText(input)
	if err != nil {
		t.Fatalf("MarkdownToPlainText returned error: %v", err)
	}

	want := "Status\n" +
		"So it's not CPU and still bootstrapping.\n" +
		"Visit Google and https://example.com.\n" +
		"First\n" +
		"Child\n" +
		"fmt.Println(\"ok\")\n" +
		"A\tB\n" +
		"1\t2\n" +
		"Alt text\n" +
		"Hello world."

	if got != want {
		t.Fatalf("plain text mismatch.\nGot:\n%s\nWant:\n%s", got, want)
	}
}

func TestMarkdownToPlainTextWithOptionsCanDisableChatBlocks(t *testing.T) {
	input := "```prompt\nHello **world**.\n```\n"

	got, err := MarkdownToPlainTextWithOptions(input, IROptions{ChatBlockIdentifiers: []string{}})
	if err != nil {
		t.Fatalf("MarkdownToPlainTextWithOptions returned error: %v", err)
	}

	want := "Hello **world**."
	if got != want {
		t.Fatalf("plain text = %q, want %q", got, want)
	}
}

func TestIRToPlainText(t *testing.T) {
	doc := IRDocument{Blocks: []IRBlock{
		{Paragraph: &IRParagraph{Segments: []IRSegment{
			{Text: "one "},
			{Text: "two", Bold: true},
		}}},
		{ThematicBreak: &IRThematicBreak{}},
		{Image: &IRImage{Alt: "diagram", Source: "diagram.png"}},
	}}

	got := IRToPlainText(doc)
	want := "one two\ndiagram"
	if got != want {
		t.Fatalf("plain text = %q, want %q", got, want)
	}
}
