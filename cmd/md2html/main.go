package main

import (
	"flag"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"

	"md2html/md2html"
)

func main() {
	outFlag := flag.String("o", "", "output file path (default: input with .html extension; use - for stdout)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: md2html [flags] input.md\n\nFlags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}

	if err := run(flag.Arg(0), *outFlag); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(inputPath, outputPath string) error {
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", inputPath, err)
	}

	content := filterHints(string(data))

	ast, err := md2html.ParseMarkdownToAST(content)
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	ir := md2html.ASTToIR(ast)

	if outputPath == "" {
		outputPath = strings.TrimSuffix(inputPath, filepath.Ext(inputPath)) + ".html"
	}

	// Rebase relative image paths from input dir to output dir.
	if outputPath != "-" {
		absInput, _ := filepath.Abs(inputPath)
		absOutput, _ := filepath.Abs(outputPath)
		rebaseImages(&ir, filepath.Dir(absInput), filepath.Dir(absOutput))
	}

	body := md2html.IRToHTML(ir)

	// Collect CSS: base + only the identifier CSS actually used.
	roles := md2html.CollectChatRoles(ir)
	css := md2html.CollectCSS(roles)

	title := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))
	fullHTML := wrapHTML(title, body, css)

	if outputPath == "-" {
		_, err := os.Stdout.WriteString(fullHTML)
		return err
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	if err := os.WriteFile(outputPath, []byte(fullHTML), 0644); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	fmt.Fprintf(os.Stderr, "%s -> %s\n", inputPath, outputPath)
	return nil
}

// rebaseImages rewrites relative image paths in the IR so they resolve
// correctly when the output HTML is in a different directory than the input.
func rebaseImages(doc *md2html.IRDocument, inputDir, outputDir string) {
	rebaseBlocks(doc.Blocks, inputDir, outputDir)
}

func rebaseBlocks(blocks []md2html.IRBlock, inputDir, outputDir string) {
	for i := range blocks {
		if blocks[i].Image != nil {
			blocks[i].Image.Source = rebasePath(blocks[i].Image.Source, inputDir, outputDir)
		}
		if blocks[i].ChatBlock != nil {
			rebaseBlocks(blocks[i].ChatBlock.Inner.Blocks, inputDir, outputDir)
		}
	}
}

func rebasePath(src, inputDir, outputDir string) string {
	if filepath.IsAbs(src) || strings.Contains(src, "://") {
		return src
	}
	abs := filepath.Join(inputDir, src)
	rel, err := filepath.Rel(outputDir, abs)
	if err != nil {
		return src
	}
	return rel
}

// filterHints removes lines that start with HINT: (case-insensitive).
func filterHints(content string) string {
	lines := strings.Split(content, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToUpper(trimmed), "HINT:") {
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

func wrapHTML(title, body, css string) string {
	var sb strings.Builder
	sb.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n")
	sb.WriteString("<meta charset=\"UTF-8\">\n")
	sb.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">\n")
	sb.WriteString(fmt.Sprintf("<title>%s</title>\n", html.EscapeString(title)))
	sb.WriteString("<style>\n")
	sb.WriteString(css)
	sb.WriteString("</style>\n")
	sb.WriteString("</head>\n<body>\n")
	sb.WriteString("<article>\n")
	sb.WriteString(body)
	sb.WriteString("</article>\n")
	sb.WriteString(zoomScript)
	sb.WriteString("</body>\n</html>\n")
	return sb.String()
}

const zoomScript = `<script>
(function() {
  var size = 100;
  document.addEventListener('keydown', function(e) {
    if ((e.metaKey || e.ctrlKey) && e.shiftKey) {
      if (e.key === '+' || e.key === '=') {
        e.preventDefault();
        size += 10;
        document.documentElement.style.fontSize = size + '%';
      } else if (e.key === '-' || e.key === '_') {
        e.preventDefault();
        size = Math.max(50, size - 10);
        document.documentElement.style.fontSize = size + '%';
      }
    }
  });
})();
</script>
`
