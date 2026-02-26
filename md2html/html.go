package md2html

import (
	"fmt"
	"html"
	"regexp"
	"strings"
	"unicode"
)

// Stage 3: IR -> HTML rendering.

// IRToHTML converts an IR document to an HTML fragment (no <html>/<body> wrapper).
func IRToHTML(doc IRDocument) string {
	var sb strings.Builder
	slugs := make(map[string]int)
	renderBlocks(&sb, doc.Blocks, slugs)
	return sb.String()
}

func renderBlocks(sb *strings.Builder, blocks []IRBlock, slugs map[string]int) {
	for _, b := range blocks {
		switch {
		case b.Heading != nil:
			renderHeading(sb, b.Heading, slugs)
		case b.Paragraph != nil:
			renderParagraph(sb, b.Paragraph)
		case b.List != nil:
			renderList(sb, b.List)
		case b.CodeBlock != nil:
			renderCodeBlock(sb, b.CodeBlock)
		case b.ChatBlock != nil:
			renderChatBlock(sb, b.ChatBlock, slugs)
		case b.Table != nil:
			renderTable(sb, b.Table)
		case b.ThematicBreak != nil:
			sb.WriteString("<hr>\n")
		case b.Image != nil:
			renderImage(sb, b.Image)
		}
	}
}

func renderHeading(sb *strings.Builder, h *IRHeading, slugs map[string]int) {
	text := segmentsToPlainText(h.Segments)
	slug := uniqueSlug(slugify(text), slugs)
	tag := fmt.Sprintf("h%d", h.Level)
	sb.WriteString("<")
	sb.WriteString(tag)
	sb.WriteString(fmt.Sprintf(" id=\"%s\">", html.EscapeString(slug)))
	sb.WriteString(fmt.Sprintf("<a href=\"#%s\">", html.EscapeString(slug)))
	renderSegments(sb, h.Segments)
	sb.WriteString("</a>")
	sb.WriteString("</")
	sb.WriteString(tag)
	sb.WriteString(">\n")
}

func renderParagraph(sb *strings.Builder, p *IRParagraph) {
	sb.WriteString("<p>")
	renderSegments(sb, p.Segments)
	sb.WriteString("</p>\n")
}

func renderList(sb *strings.Builder, l *IRList) {
	tag := "ul"
	if l.Ordered {
		tag = "ol"
	}
	sb.WriteString("<")
	sb.WriteString(tag)
	sb.WriteString(">\n")
	for _, item := range l.Items {
		sb.WriteString("<li>")
		renderSegments(sb, item.Segments)
		if item.Children != nil {
			sb.WriteString("\n")
			renderList(sb, item.Children)
		}
		sb.WriteString("</li>\n")
	}
	sb.WriteString("</")
	sb.WriteString(tag)
	sb.WriteString(">\n")
}

func renderCodeBlock(sb *strings.Builder, c *IRCodeBlock) {
	if c.Language != "" {
		sb.WriteString(fmt.Sprintf("<pre><code class=\"language-%s\">", html.EscapeString(c.Language)))
	} else {
		sb.WriteString("<pre><code>")
	}
	sb.WriteString(html.EscapeString(c.Text))
	sb.WriteString("</code></pre>\n")
}

func renderChatBlock(sb *strings.Builder, c *IRChatBlock, slugs map[string]int) {
	classes := "chat-block chat-" + html.EscapeString(c.Role)
	if !HasRoleCSS(c.Role) {
		classes += " chat-default"
	}
	sb.WriteString(fmt.Sprintf("<div class=\"%s\">\n", classes))
	sb.WriteString(fmt.Sprintf("<div class=\"chat-role\">%s</div>\n", html.EscapeString(strings.ToUpper(c.Role))))
	renderBlocks(sb, c.Inner.Blocks, slugs)
	sb.WriteString("</div>\n")
}

func renderTable(sb *strings.Builder, t *IRTable) {
	sb.WriteString("<table>\n")
	if len(t.Header) > 0 {
		sb.WriteString("<thead>\n<tr>\n")
		for _, cell := range t.Header {
			sb.WriteString("<th>")
			renderSegments(sb, cell.Segments)
			sb.WriteString("</th>\n")
		}
		sb.WriteString("</tr>\n</thead>\n")
	}
	if len(t.Rows) > 0 {
		sb.WriteString("<tbody>\n")
		for _, row := range t.Rows {
			sb.WriteString("<tr>\n")
			for _, cell := range row {
				sb.WriteString("<td>")
				renderSegments(sb, cell.Segments)
				sb.WriteString("</td>\n")
			}
			sb.WriteString("</tr>\n")
		}
		sb.WriteString("</tbody>\n")
	}
	sb.WriteString("</table>\n")
}

func renderImage(sb *strings.Builder, img *IRImage) {
	sb.WriteString(fmt.Sprintf("<img src=\"%s\" alt=\"%s\">\n",
		html.EscapeString(img.Source), html.EscapeString(img.Alt)))
}

// CollectChatRoles returns the set of chat block roles used in the document.
func CollectChatRoles(doc IRDocument) []string {
	seen := make(map[string]bool)
	var roles []string
	collectRoles(doc.Blocks, seen, &roles)
	return roles
}

func collectRoles(blocks []IRBlock, seen map[string]bool, roles *[]string) {
	for _, b := range blocks {
		if b.ChatBlock != nil {
			if !seen[b.ChatBlock.Role] {
				seen[b.ChatBlock.Role] = true
				*roles = append(*roles, b.ChatBlock.Role)
			}
			collectRoles(b.ChatBlock.Inner.Blocks, seen, roles)
		}
	}
}

func renderSegments(sb *strings.Builder, segments []IRSegment) {
	for _, seg := range segments {
		renderSegment(sb, seg)
	}
}

func renderSegment(sb *strings.Builder, seg IRSegment) {
	escaped := html.EscapeString(seg.Text)
	// Determine which tags to wrap around the text.
	// Order: link > bold > italic > code > strikethrough
	if seg.LinkURL != "" {
		sb.WriteString(fmt.Sprintf("<a href=\"%s\" target=\"_blank\">", html.EscapeString(seg.LinkURL)))
	}
	if seg.Bold {
		sb.WriteString("<strong>")
	}
	if seg.Italic {
		sb.WriteString("<em>")
	}
	if seg.Code {
		sb.WriteString("<code>")
	}
	if seg.Strikethrough {
		sb.WriteString("<del>")
	}
	sb.WriteString(escaped)
	if seg.Strikethrough {
		sb.WriteString("</del>")
	}
	if seg.Code {
		sb.WriteString("</code>")
	}
	if seg.Italic {
		sb.WriteString("</em>")
	}
	if seg.Bold {
		sb.WriteString("</strong>")
	}
	if seg.LinkURL != "" {
		sb.WriteString("</a>")
	}
}

// segmentsToPlainText extracts the concatenated plain text from segments.
func segmentsToPlainText(segments []IRSegment) string {
	var sb strings.Builder
	for _, seg := range segments {
		sb.WriteString(seg.Text)
	}
	return sb.String()
}

var collapseHyphens = regexp.MustCompile(`-{2,}`)

// slugify converts a heading string to a GitHub-style URL slug.
func slugify(s string) string {
	var sb strings.Builder
	for _, r := range strings.ToLower(s) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			sb.WriteRune(r)
		case r == ' ' || r == '-' || r == '_':
			sb.WriteByte('-')
		}
	}
	slug := collapseHyphens.ReplaceAllString(sb.String(), "-")
	slug = strings.Trim(slug, "-")
	return slug
}

// uniqueSlug returns a slug that is unique within the document. On first
// occurrence the slug is returned as-is; duplicates get a -2, -3, ... suffix.
func uniqueSlug(slug string, seen map[string]int) string {
	seen[slug]++
	if seen[slug] == 1 {
		return slug
	}
	return fmt.Sprintf("%s-%d", slug, seen[slug])
}
