# md2html

A Markdown-to-HTML converter with support for chat blocks. Designed for rendering AI conversation transcripts where prompt/agent turns are marked with fenced code block labels.

## Pipeline

```
Markdown  ->  AST  ->  IR  ->  HTML
          (goldmark)  (ir.go)  (html.go)
```

1. **AST** (`ast.go`) -- Parses markdown via [goldmark](https://github.com/yuin/goldmark) (with GFM extensions) into a normalized block AST.
2. **IR** (`ir.go`) -- Converts AST blocks into an output-format-agnostic intermediate representation. Fenced blocks tagged with chat identifiers (e.g. `prompt`, `agent`) are recursively parsed as nested markdown.
3. **HTML** (`html.go`) -- Renders the IR to a self-contained HTML document with embedded CSS and a zoom keyboard shortcut.

## Supported elements

- Headings (h1-h4) with clickable anchor links (GitHub-style slugs)
- Paragraphs, inline styles (bold, italic, code, strikethrough, links)
- Ordered and unordered lists (nested)
- GFM tables
- Fenced code blocks (with language class)
- Images
- Thematic breaks
- Chat blocks (`prompt`, `agent`, or custom identifiers)
- Unicode box-drawing tables (auto-detected and rendered as `<pre>`)

## Build

Requires Go 1.25+.

```
make
```

Runs tests, then builds and installs the binary to `~/.local/bin/md2html`. Override with:

```
make BIN=/usr/local/bin
```

## Usage

```
md2html [flags] input.md
```

### Flags

| Flag | Description |
|------|-------------|
| `-o` | Output file path. Default: input with `.html` extension. Use `-` for stdout. |

### Examples

Convert a file (writes `notes.html` next to the input):

```
md2html notes.md
```

Specify output path:

```
md2html -o output.html notes.md
```

Write to stdout:

```
md2html -o - notes.md
```

### Chat blocks

Any fenced code block **with** a language identifier is treated as a chat block. Fenced code blocks **without** a language are rendered as plain `<pre><code>`.

````markdown
```prompt
What is 2+2?
```

```agent
2+2 = **4**.
```
````

The inner content of each chat block is parsed as full markdown, so bold, lists, code blocks, and tables all work inside them. Nesting is one level deep -- fenced blocks inside chat blocks are always rendered as code.

## Language identifiers

The following identifiers have dedicated CSS styling. Any other identifier uses the default style.

| Identifier | Color             | Use case                          |
|------------|-------------------|-----------------------------------|
| `prompt`   | amber/gold        | User messages / instructions      |
| `agent`    | blue              | AI responses                      |
| `problem`  | red               | Problems / issues                 |
| `solution` | green             | Solutions / resolutions           |
| `context`  | violet            | Background context / reference    |
| `shell`    | dark, green accent | Terminal / command-line output     |
| `quote`    | teal              | Quotations                        |
| `slack`    | aubergine/plum    | Slack messages                    |
| (other)    | neutral gray      | Any unlisted identifier           |

## CSS

Styles are embedded in the output HTML. Each identifier can have a dedicated CSS file in `md2html/css/`:

| File           | Purpose                                              |
|----------------|------------------------------------------------------|
| `base.css`     | Always included. Base typography and layout.         |
| `prompt.css`   | Amber/gold theme for user prompts.                   |
| `agent.css`    | Blue theme for AI responses.                         |
| `problem.css`  | Red theme for problems.                              |
| `solution.css` | Green theme for solutions.                           |
| `context.css`  | Violet theme for context blocks.                     |
| `shell.css`    | Dark terminal theme with green accents.              |
| `quote.css`    | Teal theme for quotations.                           |
| `slack.css`    | Aubergine/plum theme for Slack messages.             |
| `default.css`  | Fallback for identifiers without a dedicated file.   |

## Test

```
make test
```

Test fixtures live in `md2html/testdata/elements/`. Each fixture has an `ast.json` input and an `ir.json` golden file. Missing golden files are auto-generated on first run.
