package md2html

import "strings"

// Stage 2: AST -> IR (generic document intermediate representation).
// The IR is output-format agnostic. Different renderers (HTML, Docs, etc.)
// consume the same IR.

// IRDocument is the top-level IR container.
type IRDocument struct {
	Blocks []IRBlock
}

// IRBlock is a sum type. Exactly one field is non-nil.
type IRBlock struct {
	Paragraph     *IRParagraph     `json:"paragraph,omitempty"`
	Heading       *IRHeading       `json:"heading,omitempty"`
	List          *IRList          `json:"list,omitempty"`
	CodeBlock     *IRCodeBlock     `json:"codeBlock,omitempty"`
	ChatBlock     *IRChatBlock     `json:"chatBlock,omitempty"`
	Table         *IRTable         `json:"table,omitempty"`
	ThematicBreak *IRThematicBreak `json:"thematicBreak,omitempty"`
	Image         *IRImage         `json:"image,omitempty"`
}

type IRParagraph struct {
	Segments []IRSegment `json:"segments"`
}

type IRHeading struct {
	Level    int         `json:"level"`
	Segments []IRSegment `json:"segments"`
}

type IRList struct {
	Ordered bool         `json:"ordered"`
	Items   []IRListItem `json:"items"`
}

type IRListItem struct {
	Segments []IRSegment `json:"segments"`
	Children *IRList     `json:"children,omitempty"`
}

type IRCodeBlock struct {
	Language string `json:"language,omitempty"`
	Text     string `json:"text"`
}

type IRChatBlock struct {
	Role  string     `json:"role"`
	Inner IRDocument `json:"inner"`
}

type IRTable struct {
	Header []IRTableCell   `json:"header"`
	Rows   [][]IRTableCell `json:"rows"`
}

type IRTableCell struct {
	Segments []IRSegment `json:"segments"`
}

type IRThematicBreak struct{}

type IRImage struct {
	Alt    string `json:"alt"`
	Source string `json:"source"`
}

type IRSegment struct {
	Text          string `json:"text"`
	Bold          bool   `json:"bold,omitempty"`
	Italic        bool   `json:"italic,omitempty"`
	Code          bool   `json:"code,omitempty"`
	Strikethrough bool   `json:"strikethrough,omitempty"`
	LinkURL       string `json:"linkUrl,omitempty"`
}

// ASTToIR converts an AST Document to a generic IR Document.
// Fenced code blocks with a language identifier are treated as chat blocks
// whose content is recursively parsed as markdown. Fenced code blocks
// without a language remain as plain code blocks.
func ASTToIR(doc Document) IRDocument {
	return IRDocument{
		Blocks: convertBlocks(doc.Blocks, 0),
	}
}

func convertBlocks(blocks []Block, depth int) []IRBlock {
	out := make([]IRBlock, 0, len(blocks))
	i := 0
	for i < len(blocks) {
		b := blocks[i]
		switch b.Kind {
		case Paragraph:
			if containsBoxDrawing(b.Text) {
				out = append(out, IRBlock{CodeBlock: &IRCodeBlock{Text: b.Text}})
			} else {
				out = append(out, IRBlock{Paragraph: &IRParagraph{
					Segments: inlineStylesToSegments(b.Text, b.InlineStyles),
				}})
			}
		case Heading1:
			out = append(out, IRBlock{Heading: &IRHeading{Level: 1, Segments: inlineStylesToSegments(b.Text, b.InlineStyles)}})
		case Heading2:
			out = append(out, IRBlock{Heading: &IRHeading{Level: 2, Segments: inlineStylesToSegments(b.Text, b.InlineStyles)}})
		case Heading3:
			out = append(out, IRBlock{Heading: &IRHeading{Level: 3, Segments: inlineStylesToSegments(b.Text, b.InlineStyles)}})
		case Heading4:
			out = append(out, IRBlock{Heading: &IRHeading{Level: 4, Segments: inlineStylesToSegments(b.Text, b.InlineStyles)}})
		case ListItem:
			// Consume consecutive list items and group into nested IRList.
			list, consumed := groupListItems(blocks[i:])
			out = append(out, IRBlock{List: list})
			i += consumed
			continue
		case CodeBlock:
			lang := b.CodeLanguage
			if lang != "" && depth == 0 {
				// Recursively parse the code block content as markdown.
				// Fence box-drawing lines so goldmark preserves them.
				innerAST, err := ParseMarkdownToAST(fenceBoxDrawing(b.Text))
				if err == nil {
					innerIR := IRDocument{Blocks: convertBlocks(innerAST.Blocks, depth+1)}
					out = append(out, IRBlock{ChatBlock: &IRChatBlock{Role: lang, Inner: innerIR}})
				} else {
					out = append(out, IRBlock{CodeBlock: &IRCodeBlock{Text: b.Text}})
				}
			} else {
				out = append(out, IRBlock{CodeBlock: &IRCodeBlock{Text: b.Text}})
			}
		case Table:
			if b.Table != nil && len(b.Table.Rows) > 0 {
				irTable := convertTable(b.Table)
				out = append(out, IRBlock{Table: irTable})
			}
		case ThematicBreak:
			out = append(out, IRBlock{ThematicBreak: &IRThematicBreak{}})
		case Image:
			if b.Image != nil {
				out = append(out, IRBlock{Image: &IRImage{Alt: b.Image.Alt, Source: b.Image.Source}})
			}
		}
		i++
	}
	return out
}

// groupListItems consumes consecutive ListItem blocks starting at blocks[0]
// and returns a nested IRList tree. Returns the IRList and how many blocks
// were consumed.
func groupListItems(blocks []Block) (*IRList, int) {
	all := 0
	for all < len(blocks) && blocks[all].Kind == ListItem {
		all++
	}
	items := blocks[:all]

	if root := buildInterleavedOrderedList(items); root != nil {
		return root, all
	}

	consumed := all
	for i := 1; i < all; i++ {
		if startsNewTopLevelList(items[i-1], items[i]) {
			consumed = i
			break
		}
	}

	root := buildListTree(items[:consumed], 0)
	return root, consumed
}

func startsNewTopLevelList(prev, cur Block) bool {
	if cur.ListDepth != 0 || prev.ListDepth != 0 {
		return false
	}
	return prev.Ordered != cur.Ordered
}

// buildInterleavedOrderedList folds this common LLM output pattern:
// ordered item, top-level bullets, ordered item, top-level bullets...
// into one ordered list with per-item unordered children.
func buildInterleavedOrderedList(items []Block) *IRList {
	if len(items) < 3 || !items[0].Ordered {
		return nil
	}
	for _, it := range items {
		if it.ListDepth != 0 {
			return nil
		}
	}

	hasUnordered := false
	hasOrderedAfterUnordered := false
	seenUnordered := false
	for i := range items {
		if !items[i].Ordered {
			hasUnordered = true
			seenUnordered = true
			continue
		}
		if seenUnordered {
			hasOrderedAfterUnordered = true
		}
	}
	if !hasUnordered || !hasOrderedAfterUnordered {
		return nil
	}

	root := &IRList{Ordered: true, Items: make([]IRListItem, 0)}
	i := 0
	for i < len(items) {
		if !items[i].Ordered {
			return nil
		}
		parent := IRListItem{
			Segments: inlineStylesToSegments(items[i].Text, items[i].InlineStyles),
		}
		i++

		start := i
		for i < len(items) && !items[i].Ordered {
			i++
		}
		if i > start {
			children := &IRList{Ordered: false, Items: make([]IRListItem, 0, i-start)}
			for j := start; j < i; j++ {
				children.Items = append(children.Items, IRListItem{
					Segments: inlineStylesToSegments(items[j].Text, items[j].InlineStyles),
				})
			}
			parent.Children = children
		}
		root.Items = append(root.Items, parent)
	}

	return root
}

// buildListTree builds a nested IRList from flat ListItem blocks at the
// given minimum depth.
func buildListTree(items []Block, minDepth int) *IRList {
	if len(items) == 0 {
		return nil
	}

	list := &IRList{
		Ordered: items[0].Ordered,
		Items:   make([]IRListItem, 0),
	}

	i := 0
	for i < len(items) {
		if items[i].ListDepth == minDepth {
			item := IRListItem{
				Segments: inlineStylesToSegments(items[i].Text, items[i].InlineStyles),
			}
			// Collect children at deeper depths.
			j := i + 1
			for j < len(items) && items[j].ListDepth > minDepth {
				j++
			}
			if j > i+1 {
				item.Children = buildListTree(items[i+1:j], minDepth+1)
			}
			list.Items = append(list.Items, item)
			i = j
		} else {
			// Items at deeper depth without a parent -- still group them.
			j := i + 1
			for j < len(items) && items[j].ListDepth > minDepth {
				j++
			}
			child := buildListTree(items[i:j], items[i].ListDepth)
			if child != nil && len(list.Items) > 0 {
				lastItem := &list.Items[len(list.Items)-1]
				lastItem.Children = child
			} else if child != nil {
				// No parent item -- wrap in a dummy item.
				list.Items = append(list.Items, IRListItem{Children: child})
			}
			i = j
		}
	}

	return list
}

func convertTable(t *TableData) *IRTable {
	if t == nil || len(t.Rows) == 0 {
		return nil
	}
	irTable := &IRTable{}
	// First row is header.
	irTable.Header = convertTableRow(t.Rows[0])
	irTable.Rows = make([][]IRTableCell, 0, len(t.Rows)-1)
	for _, row := range t.Rows[1:] {
		irTable.Rows = append(irTable.Rows, convertTableRow(row))
	}
	return irTable
}

func convertTableRow(cells []TableCell) []IRTableCell {
	out := make([]IRTableCell, len(cells))
	for i, cell := range cells {
		out[i] = IRTableCell{
			Segments: inlineStylesToSegments(cell.Text, cell.InlineStyles),
		}
	}
	return out
}

// containsBoxDrawing reports whether s contains any Unicode Box Drawing
// character (U+2500 .. U+257F). Paragraphs with these characters are
// preformatted tables that must be rendered as code blocks.
func containsBoxDrawing(s string) bool {
	for _, r := range s {
		if r >= 0x2500 && r <= 0x257F {
			return true
		}
	}
	return false
}

// fenceBoxDrawing wraps contiguous runs of lines that contain box-drawing
// characters in fenced code blocks (```) so that goldmark preserves them
// as preformatted text instead of mangling them into paragraphs or tables.
func fenceBoxDrawing(text string) string {
	lines := strings.Split(text, "\n")
	var sb strings.Builder
	inBox := false
	for _, line := range lines {
		if containsBoxDrawing(line) {
			if !inBox {
				sb.WriteString("```\n")
				inBox = true
			}
			sb.WriteString(line)
			sb.WriteByte('\n')
		} else {
			if inBox {
				sb.WriteString("```\n")
				inBox = false
			}
			sb.WriteString(line)
			sb.WriteByte('\n')
		}
	}
	if inBox {
		sb.WriteString("```\n")
	}
	return strings.TrimRight(sb.String(), "\n")
}

// inlineStylesToSegments converts a plain text string with InlineStyle spans
// into a list of IRSegments, each carrying its own style attributes.
func inlineStylesToSegments(text string, styles []InlineStyle) []IRSegment {
	if text == "" {
		return nil
	}
	runes := []rune(text)
	if len(styles) == 0 {
		return []IRSegment{{Text: text}}
	}

	segments := make([]IRSegment, 0)
	pos := 0
	for _, s := range styles {
		start := s.Start
		end := s.End
		if start > len(runes) {
			start = len(runes)
		}
		if end > len(runes) {
			end = len(runes)
		}
		// Emit unstyled text before this span.
		if pos < start {
			segments = append(segments, IRSegment{Text: string(runes[pos:start])})
		}
		if start < end {
			segments = append(segments, IRSegment{
				Text:          string(runes[start:end]),
				Bold:          s.Bold,
				Italic:        s.Italic,
				Code:          s.Code,
				Strikethrough: s.Strikethrough,
				LinkURL:       s.LinkURL,
			})
		}
		pos = end
	}
	// Emit trailing unstyled text.
	if pos < len(runes) {
		segments = append(segments, IRSegment{Text: string(runes[pos:])})
	}
	return segments
}
