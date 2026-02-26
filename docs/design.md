# Design: md2html CLI

## Architecture: 3-Stage Pipeline

```
.md -> AST -> IR -> HTML
        ^      ^      ^
    Stage 1  Stage 2  Stage 3
```

All three stages live in the `md2html` package, which is the reusable core. google-docs-skill can later import it and replace Stage 3 with a Docs renderer.

### Package structure

```
md2html/
  go.mod
  ast.go               -- Stage 1: .md -> AST (goldmark parser)
  ir.go                -- Stage 2: AST -> IR (generic document IR)
  html.go              -- Stage 3: IR -> HTML renderer
  css.go               -- embedded CSS (base + per-identifier + default)
  css/                  -- source CSS files, embedded via go:embed
    base.css
    prompt.css
    agent.css
    default.css        -- fallback for unrecognized identifiers
  *_test.go
  testdata/
  cmd/md2html/
    main.go            -- CLI: one .md in, one .html out
  docs/
    design.md
    project.md
```

## CLI interface

```
md2html input.md                    # writes input.html next to input.md
md2html -o output.html input.md     # explicit output path
md2html -o - input.md               # write to stdout
```

One .md file in, one self-contained .html file out. No directory scanning, no index generation.

## CSS strategy

### Embedded CSS

All CSS is embedded in the binary via `go:embed`. The `css/` directory contains:

| File          | Purpose                                                    |
|---------------|------------------------------------------------------------|
| `base.css`    | Layout, typography, tables, code blocks, `.chat-block` base |
| `prompt.css`  | `.chat-prompt` -- blue left border, light blue background  |
| `agent.css`   | `.chat-agent` -- green left border, light green background |
| `default.css` | `.chat-default` -- gray left border, neutral background    |

### How CSS is included in output

The output HTML contains `<style>` blocks (not `<link>` tags) so each file is fully self-contained. The renderer:

1. Always includes `base.css`.
2. Scans the IR for chat blocks and collects the set of roles used.
3. For each role, includes `{role}.css` if it exists in the embedded FS, otherwise includes `default.css` with `.chat-default` as the fallback class.

This means:
- Known identifiers (`prompt`, `agent`) get their dedicated CSS.
- Any new/unknown identifier automatically gets the default styling.
- To add a new styled identifier, just add a `{name}.css` file in `css/` and rebuild.

### Default identifier handling

When the HTML renderer encounters a chat block with role `foo` and no `foo.css` exists:
- The `<div>` gets `class="chat-block chat-foo chat-default"`.
- `default.css` styles `.chat-default` as the fallback.
- If `foo.css` is later added, it styles `.chat-foo` and takes precedence.

## Chat block rendering

Custom code blocks (language identifier in a configurable list) contain inner markdown:

- **Custom identifier** -> recursively parse inner text as markdown -> render nested IR to HTML -> wrap in `<div class="chat-block chat-{role}">`
- **Programming language** -> render as `<pre><code class="language-{lang}">`
- **No identifier** -> render as `<pre><code>`

## Other rendering rules (from PROMPT.md)

- Chat blocks use fixed-width font (via `.chat-block` base CSS).
- All `<a>` tags get `target="_blank"`.
- Inline `<script>` for CMD+SHIFT++/- zoom (adjusts `html.style.fontSize`).
- `HINT:` lines in markdown are stripped (not part of content).
