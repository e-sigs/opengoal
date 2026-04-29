BIN      ?= og
PREFIX   ?= $(HOME)/.local
BINDIR   ?= $(PREFIX)/bin
OPENCODE ?= $(HOME)/.config/opencode

GO       ?= go
PKG      := ./cmd/og

.PHONY: all build install install-bin install-opencode uninstall clean test help

all: build

## build         — compile the og binary into ./og
build:
	$(GO) build -o $(BIN) $(PKG)

## install       — build, then install the binary and OpenCode integration
install: install-bin install-opencode
	@echo
	@echo "✅ opengoal installed."
	@echo
	@echo "Binary:        $(BINDIR)/$(BIN)"
	@echo "Agents:        $(OPENCODE)/agents/{worker,reviewer,orchestrator}.md"
	@echo "Slash cmds:    $(OPENCODE)/commands/og*.md, task-*.md, today.md"
	@echo
	@echo "Make sure $(BINDIR) is on your PATH, then run:"
	@echo "    og help"
	@echo "    og list-create my-first-roadmap"
	@echo "    og today"
	@echo

## install-bin   — only install the binary into $(BINDIR)
install-bin: build
	@mkdir -p $(BINDIR)
	@install -m 0755 $(BIN) $(BINDIR)/$(BIN)
	@echo "→ installed $(BINDIR)/$(BIN)"

## install-opencode — only copy agents and slash commands into $(OPENCODE)
install-opencode:
	@mkdir -p $(OPENCODE)/agents $(OPENCODE)/commands
	@cp install/agents/*.md    $(OPENCODE)/agents/
	@cp install/commands/*.md  $(OPENCODE)/commands/
	@echo "→ installed agents into $(OPENCODE)/agents/"
	@echo "→ installed slash commands into $(OPENCODE)/commands/"

## uninstall     — remove the binary and OpenCode files
uninstall:
	@rm -f $(BINDIR)/$(BIN)
	@rm -f $(OPENCODE)/agents/worker.md
	@rm -f $(OPENCODE)/agents/reviewer.md
	@rm -f $(OPENCODE)/agents/orchestrator.md
	@for f in install/commands/*.md; do rm -f $(OPENCODE)/commands/$$(basename $$f); done
	@echo "→ uninstalled (data files at ~/.local/share/opencode/goals* are preserved)"

## test          — go vet + go build sanity check
test:
	$(GO) vet $(PKG)
	$(GO) build -o /dev/null $(PKG)

## clean         — remove the local ./og build artifact
clean:
	@rm -f $(BIN)

## help          — show this help
help:
	@grep -hE '^##' $(MAKEFILE_LIST) | sed 's/## //'
