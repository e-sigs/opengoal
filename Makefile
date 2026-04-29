BIN      ?= og
PREFIX   ?= $(HOME)/.local
BINDIR   ?= $(PREFIX)/bin
# NOTE: do not name this OPENCODE — opencode itself exports OPENCODE=1 in
# child processes, which would shadow this variable when `make` is run from
# inside an opencode session.
OPENCODE_DIR ?= $(HOME)/.config/opencode

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
	@echo "Agents:        $(OPENCODE_DIR)/agents/{worker,reviewer,orchestrator}.md"
	@echo "Slash cmds:    $(OPENCODE_DIR)/commands/og*.md, task-*.md, today.md"
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

## install-opencode — only copy agents and slash commands into $(OPENCODE_DIR)
install-opencode:
	@mkdir -p $(OPENCODE_DIR)/agents $(OPENCODE_DIR)/commands
	@cp install/agents/*.md    $(OPENCODE_DIR)/agents/
	@cp install/commands/*.md  $(OPENCODE_DIR)/commands/
	@echo "→ installed agents into $(OPENCODE_DIR)/agents/"
	@echo "→ installed slash commands into $(OPENCODE_DIR)/commands/"

## uninstall     — remove the binary and OpenCode files
uninstall:
	@rm -f $(BINDIR)/$(BIN)
	@rm -f $(OPENCODE_DIR)/agents/worker.md
	@rm -f $(OPENCODE_DIR)/agents/reviewer.md
	@rm -f $(OPENCODE_DIR)/agents/orchestrator.md
	@for f in install/commands/*.md; do rm -f $(OPENCODE_DIR)/commands/$$(basename $$f); done
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
