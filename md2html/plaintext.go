package md2html

import "strings"

// MarkdownToPlainText parses markdown and returns its user-visible plain text.
func MarkdownToPlainText(markdown string) (string, error) {
	return MarkdownToPlainTextWithOptions(markdown, IROptions{})
}

// MarkdownToPlainTextWithOptions parses markdown with explicit IR options and
// returns its user-visible plain text.
func MarkdownToPlainTextWithOptions(markdown string, opts IROptions) (string, error) {
	ast, err := ParseMarkdownToAST(markdown)
	if err != nil {
		return "", err
	}
	return IRToPlainText(ASTToIRWithOptions(ast, opts)), nil
}

// IRToPlainText converts an IR document to a plain-text representation.
func IRToPlainText(doc IRDocument) string {
	return strings.TrimSpace(strings.Join(plainBlocks(doc.Blocks), "\n"))
}

func plainBlocks(blocks []IRBlock) []string {
	lines := make([]string, 0, len(blocks))
	for _, b := range blocks {
		switch {
		case b.Heading != nil:
			appendPlainLine(&lines, segmentsToPlainText(b.Heading.Segments))
		case b.Paragraph != nil:
			appendPlainLine(&lines, segmentsToPlainText(b.Paragraph.Segments))
		case b.List != nil:
			appendPlainList(&lines, b.List)
		case b.CodeBlock != nil:
			appendPlainLine(&lines, strings.TrimRight(b.CodeBlock.Text, "\n"))
		case b.ChatBlock != nil:
			lines = append(lines, plainBlocks(b.ChatBlock.Inner.Blocks)...)
		case b.Table != nil:
			appendPlainTable(&lines, b.Table)
		case b.Image != nil:
			appendPlainLine(&lines, b.Image.Alt)
		}
	}
	return lines
}

func appendPlainList(lines *[]string, list *IRList) {
	if list == nil {
		return
	}
	for _, item := range list.Items {
		appendPlainLine(lines, segmentsToPlainText(item.Segments))
		if item.Children != nil {
			appendPlainList(lines, item.Children)
		}
	}
}

func appendPlainTable(lines *[]string, table *IRTable) {
	if table == nil {
		return
	}
	appendPlainTableRow(lines, table.Header)
	for _, row := range table.Rows {
		appendPlainTableRow(lines, row)
	}
}

func appendPlainTableRow(lines *[]string, cells []IRTableCell) {
	if len(cells) == 0 {
		return
	}
	parts := make([]string, len(cells))
	for i, cell := range cells {
		parts[i] = segmentsToPlainText(cell.Segments)
	}
	appendPlainLine(lines, strings.Join(parts, "\t"))
}

func appendPlainLine(lines *[]string, text string) {
	if strings.TrimSpace(text) == "" {
		return
	}
	*lines = append(*lines, text)
}
