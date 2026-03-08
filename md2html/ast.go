package md2html

import (
	"strings"
	"unicode"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	extast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

// Stage 1: Markdown text -> normalized AST document.
// This stage is pure parsing/normalization with no output-format concerns.

type BlockKind string

const (
	Paragraph     BlockKind = "paragraph"
	Heading1      BlockKind = "heading_1"
	Heading2      BlockKind = "heading_2"
	Heading3      BlockKind = "heading_3"
	Heading4      BlockKind = "heading_4"
	ListItem      BlockKind = "list_item"
	CodeBlock     BlockKind = "code_block"
	Table         BlockKind = "table"
	Image         BlockKind = "image"
	ThematicBreak BlockKind = "thematic_break"
)

type Document struct {
	Blocks []Block
}

type Block struct {
	Kind         BlockKind      `json:"kind"`
	Text         string         `json:"text,omitempty"`
	CodeLanguage string         `json:"codeLanguage,omitempty"`
	ListDepth    int            `json:"listDepth,omitempty"`
	Ordered      bool           `json:"ordered,omitempty"`
	BlankBefore  bool           `json:"blankBefore,omitempty"`
	Table        *TableData     `json:"table,omitempty"`
	Image        *ImageData     `json:"image,omitempty"`
	InlineStyles []InlineStyle  `json:"inlineStyles,omitempty"`
}

type ImageData struct {
	Alt    string `json:"alt"`
	Source string `json:"source"`
}

type TableData struct {
	Columns int          `json:"columns"`
	Rows    [][]TableCell `json:"rows"`
}

type TableCell struct {
	Text         string        `json:"text"`
	InlineStyles []InlineStyle `json:"inlineStyles,omitempty"`
}

type InlineStyle struct {
	Start         int    `json:"start"`
	End           int    `json:"end"`
	Bold          bool   `json:"bold,omitempty"`
	Italic        bool   `json:"italic,omitempty"`
	Code          bool   `json:"code,omitempty"`
	Strikethrough bool   `json:"strikethrough,omitempty"`
	LinkURL       string `json:"linkUrl,omitempty"`
}

func (s InlineStyle) hasValue() bool {
	return s.Bold || s.Italic || s.Code || s.Strikethrough || strings.TrimSpace(s.LinkURL) != ""
}

// inlineSegment is an internal type used during text+style extraction.
type inlineSegment struct {
	text  string
	style InlineStyle
}

// ParseMarkdownToAST parses markdown text into a normalized AST Document.
func ParseMarkdownToAST(markdown string) (Document, error) {
	source := []byte(markdown)
	reader := text.NewReader(source)
	doc := goldmark.New(goldmark.WithExtensions(extension.GFM)).Parser().Parse(reader)

	blocks := make([]Block, 0)
	for node := doc.FirstChild(); node != nil; node = node.NextSibling() {
		prevLen := len(blocks)
		switch n := node.(type) {
		case *ast.Paragraph:
			if imageBlock, ok := newImageBlockFromParagraph(n, source); ok {
				appendIfNotEmpty(&blocks, imageBlock)
			} else {
				appendIfNotEmpty(&blocks, newTextBlock(Paragraph, n, source))
			}
		case *ast.Heading:
			kind := headingKind(n.Level)
			appendIfNotEmpty(&blocks, newTextBlock(kind, n, source))
		case *ast.FencedCodeBlock:
			appendIfNotEmpty(&blocks, newCodeBlock(n.Lines(), source, fencedCodeBlockLanguage(n, source)))
		case *ast.CodeBlock:
			appendIfNotEmpty(&blocks, newCodeBlock(n.Lines(), source, ""))
		case *ast.List:
			appendListItems(&blocks, n, source, 0, n.IsOrdered())
		case *extast.Table:
			appendIfNotEmpty(&blocks, newTableBlock(n, source))
		case *ast.ThematicBreak:
			blocks = append(blocks, Block{Kind: ThematicBreak})
		default:
			appendIfNotEmpty(&blocks, newTextBlock(Paragraph, n, source))
		}
		// Preserve blank lines from the markdown source.
		if len(blocks) > prevLen && prevLen > 0 && node.HasBlankPreviousLines() {
			if !isHeadingKind(blocks[prevLen].Kind) && !isHeadingKind(blocks[prevLen-1].Kind) {
				blocks[prevLen].BlankBefore = true
			}
		}
	}

	return Document{Blocks: blocks}, nil
}

func headingKind(level int) BlockKind {
	switch level {
	case 1:
		return Heading1
	case 2:
		return Heading2
	case 3:
		return Heading3
	case 4:
		return Heading4
	default:
		return Paragraph
	}
}

func isHeadingKind(kind BlockKind) bool {
	return kind == Heading1 || kind == Heading2 || kind == Heading3 || kind == Heading4
}

func appendIfNotEmpty(out *[]Block, block Block) {
	if block.Kind == Image {
		if block.Image == nil || strings.TrimSpace(block.Image.Source) == "" {
			return
		}
		*out = append(*out, block)
		return
	}
	if block.Kind == Table {
		if block.Table == nil || block.Table.Columns == 0 || len(block.Table.Rows) == 0 {
			return
		}
		*out = append(*out, block)
		return
	}
	if strings.TrimSpace(block.Text) == "" {
		return
	}
	*out = append(*out, block)
}

func appendListItems(out *[]Block, list *ast.List, source []byte, depth int, ordered bool) {
	for item := list.FirstChild(); item != nil; item = item.NextSibling() {
		listItem, ok := item.(*ast.ListItem)
		if !ok {
			continue
		}
		for child := listItem.FirstChild(); child != nil; child = child.NextSibling() {
			switch c := child.(type) {
			case *ast.Paragraph:
				appendIfNotEmpty(out, newListItemBlock(c, source, depth, ordered))
			case *ast.List:
				appendListItems(out, c, source, depth+1, c.IsOrdered())
			default:
				appendIfNotEmpty(out, newListItemBlock(c, source, depth, ordered))
			}
		}
	}
}

func newTextBlock(kind BlockKind, node ast.Node, source []byte) Block {
	textValue, styles := extractNodeTextAndStyles(node, source)
	return Block{
		Kind:         kind,
		Text:         textValue,
		InlineStyles: styles,
	}
}

func newListItemBlock(node ast.Node, source []byte, depth int, ordered bool) Block {
	block := newTextBlock(ListItem, node, source)
	block.ListDepth = depth
	block.Ordered = ordered
	return block
}

func newCodeBlock(lines *text.Segments, source []byte, codeLanguage string) Block {
	textValue := extractBlockText(lines, source)
	return Block{
		Kind:         CodeBlock,
		Text:         textValue,
		CodeLanguage: strings.ToLower(strings.TrimSpace(codeLanguage)),
	}
}

func fencedCodeBlockLanguage(node *ast.FencedCodeBlock, source []byte) string {
	if node == nil || node.Info == nil {
		return ""
	}
	info := strings.TrimSpace(string(node.Info.Text(source)))
	if info == "" {
		return ""
	}
	parts := strings.Fields(info)
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

func newTableBlock(node *extast.Table, source []byte) Block {
	table := &TableData{Rows: make([][]TableCell, 0)}
	maxColumns := 0

	for row := node.FirstChild(); row != nil; row = row.NextSibling() {
		switch row.(type) {
		case *extast.TableHeader, *extast.TableRow:
		default:
			continue
		}

		cells := extractTableRow(row, source)
		if len(cells) == 0 {
			continue
		}
		if len(cells) > maxColumns {
			maxColumns = len(cells)
		}
		table.Rows = append(table.Rows, cells)
	}

	if maxColumns == 0 || len(table.Rows) == 0 {
		return Block{Kind: Table}
	}

	for i := range table.Rows {
		for len(table.Rows[i]) < maxColumns {
			table.Rows[i] = append(table.Rows[i], TableCell{})
		}
	}
	table.Columns = maxColumns

	return Block{Kind: Table, Table: table}
}

func newImageBlockFromParagraph(node *ast.Paragraph, source []byte) (Block, bool) {
	if node == nil || node.FirstChild() == nil || node.FirstChild() != node.LastChild() {
		return Block{}, false
	}
	imageNode, ok := node.FirstChild().(*ast.Image)
	if !ok {
		return Block{}, false
	}

	altText, _ := extractNodeTextAndStyles(imageNode, source)
	sourceURI := strings.TrimSpace(string(imageNode.Destination))
	if sourceURI == "" {
		return Block{}, false
	}

	return Block{
		Kind:  Image,
		Image: &ImageData{Alt: altText, Source: sourceURI},
	}, true
}

func extractTableRow(row ast.Node, source []byte) []TableCell {
	cells := make([]TableCell, 0)
	for cell := row.FirstChild(); cell != nil; cell = cell.NextSibling() {
		tableCell, ok := cell.(*extast.TableCell)
		if !ok {
			continue
		}
		textValue, styles := extractNodeTextAndStyles(tableCell, source)
		cells = append(cells, TableCell{
			Text:         textValue,
			InlineStyles: styles,
		})
	}
	return cells
}

func extractBlockText(lines *text.Segments, source []byte) string {
	if lines == nil || lines.Len() == 0 {
		return ""
	}
	var sb strings.Builder
	for i := 0; i < lines.Len(); i++ {
		segment := lines.At(i)
		sb.Write(segment.Value(source))
	}
	return strings.TrimRight(sb.String(), "\n")
}

func extractNodeTextAndStyles(node ast.Node, source []byte) (string, []InlineStyle) {
	segments := trimInlineSegments(extractInlineSegments(node, source))
	if len(segments) == 0 {
		return "", nil
	}

	var sb strings.Builder
	var spans []InlineStyle
	offset := 0
	for _, seg := range segments {
		if seg.text == "" {
			continue
		}
		sb.WriteString(seg.text)
		segLen := len([]rune(seg.text))
		if seg.style.hasValue() && segLen > 0 {
			span := seg.style
			span.Start = offset
			span.End = offset + segLen
			appendStyleSpan(&spans, span)
		}
		offset += segLen
	}
	return sb.String(), spans
}

func extractInlineSegments(node ast.Node, source []byte) []inlineSegment {
	out := make([]inlineSegment, 0)

	var walk func(ast.Node, InlineStyle)
	walk = func(n ast.Node, style InlineStyle) {
		switch t := n.(type) {
		case *ast.Text:
			textValue := string(t.Segment.Value(source))
			if t.SoftLineBreak() || t.HardLineBreak() {
				textValue += " "
			}
			appendInlineSegment(&out, inlineSegment{text: textValue, style: style})
		case *ast.CodeSpan:
			nextStyle := style
			nextStyle.Code = true
			appendInlineSegment(&out, inlineSegment{text: string(t.Text(source)), style: nextStyle})
		case *ast.Emphasis:
			nextStyle := style
			if t.Level >= 2 {
				nextStyle.Bold = true
			} else {
				nextStyle.Italic = true
			}
			for child := t.FirstChild(); child != nil; child = child.NextSibling() {
				walk(child, nextStyle)
			}
		case *extast.Strikethrough:
			nextStyle := style
			nextStyle.Strikethrough = true
			for child := t.FirstChild(); child != nil; child = child.NextSibling() {
				walk(child, nextStyle)
			}
		case *ast.Link:
			nextStyle := style
			nextStyle.LinkURL = string(t.Destination)
			for child := t.FirstChild(); child != nil; child = child.NextSibling() {
				walk(child, nextStyle)
			}
		case *ast.AutoLink:
			nextStyle := style
			nextStyle.LinkURL = string(t.URL(source))
			appendInlineSegment(&out, inlineSegment{
				text:  string(t.Label(source)),
				style: nextStyle,
			})
		default:
			for child := n.FirstChild(); child != nil; child = child.NextSibling() {
				walk(child, style)
			}
		}
	}

	walk(node, InlineStyle{})
	return out
}

func trimInlineSegments(segments []inlineSegment) []inlineSegment {
	if len(segments) == 0 {
		return nil
	}

	start := 0
	for start < len(segments) {
		trimmed := strings.TrimLeftFunc(segments[start].text, unicode.IsSpace)
		if trimmed == "" {
			start++
			continue
		}
		segments[start].text = trimmed
		break
	}
	if start == len(segments) {
		return nil
	}

	end := len(segments) - 1
	for end >= start {
		trimmed := strings.TrimRightFunc(segments[end].text, unicode.IsSpace)
		if trimmed == "" {
			end--
			continue
		}
		segments[end].text = trimmed
		break
	}
	if end < start {
		return nil
	}

	trimmed := make([]inlineSegment, 0, end-start+1)
	for i := start; i <= end; i++ {
		if segments[i].text == "" {
			continue
		}
		trimmed = append(trimmed, segments[i])
	}
	return trimmed
}

func appendInlineSegment(out *[]inlineSegment, seg inlineSegment) {
	if seg.text == "" {
		return
	}
	if n := len(*out); n > 0 {
		last := &(*out)[n-1]
		if last.style == seg.style {
			last.text += seg.text
			return
		}
	}
	*out = append(*out, seg)
}

func appendStyleSpan(spans *[]InlineStyle, span InlineStyle) {
	if span.End <= span.Start {
		return
	}
	if n := len(*spans); n > 0 {
		last := &(*spans)[n-1]
		if stylesEqual(*last, span) && last.End == span.Start {
			last.End = span.End
			return
		}
	}
	*spans = append(*spans, span)
}

func stylesEqual(a, b InlineStyle) bool {
	return a.Bold == b.Bold && a.Italic == b.Italic && a.Code == b.Code &&
		a.Strikethrough == b.Strikethrough && a.LinkURL == b.LinkURL
}

