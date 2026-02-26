# md2html

## Problem

Agent-history repo contains `.md` files recording chat sessions between a user and AI agents. These need to be converted to styled, self-contained `.html` pages.

Using an AI agent to do the conversion is slow and expensive. This is a deterministic transformation -- it should be a CLI.

## Parallel

`$CODE/skills/google-docs-skill` solves the same class of problem for Google Docs: markdown in, rich document out. Its three-stage pipeline (`.md -> AST -> IR -> Docs requests`) proves the approach. Stage 1 and the IR concept are format-agnostic.

## Goal

A CLI (`md2html`) that takes one `.md` file and outputs one self-contained `.html` file with embedded CSS.

## Source repo

`$CODE/md2html/`

## Package layout

- `*.go` at module root -- reusable core (AST parser, IR, HTML renderer, embedded CSS)
- `cmd/md2html/` -- CLI entry point

## Usage

```
md2html input.md                    # -> input.html
md2html -o out.html input.md        # explicit output
md2html -o - input.md               # stdout
```

## CSS strategy

- All CSS embedded in the binary via `go:embed`
- `base.css` always included; identifier CSS included only for roles used
- `default.css` is fallback for unrecognized identifiers
- To add a new styled identifier: add `{name}.css` in `css/` and rebuild
