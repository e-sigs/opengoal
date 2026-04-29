#!/usr/bin/env bash
# install.sh — install opengoal binary + OpenCode agents/commands.
# Equivalent to `make install`. Use this if you don't have make.
#
# Env overrides:
#   PREFIX     install prefix for the binary (default: $HOME/.local)
#   BINDIR     binary dir (default: $PREFIX/bin)
#   OPENCODE_DIR  opencode config dir (default: $HOME/.config/opencode)
#
# NOTE: do not name this OPENCODE — opencode itself exports OPENCODE=1 in
# child processes, which would shadow this variable when install.sh is run
# from inside an opencode session.

set -euo pipefail

PREFIX="${PREFIX:-$HOME/.local}"
BINDIR="${BINDIR:-$PREFIX/bin}"
OPENCODE_DIR="${OPENCODE_DIR:-$HOME/.config/opencode}"
BIN="og"

repo_root="$(cd "$(dirname "$0")" && pwd)"
cd "$repo_root"

if ! command -v go >/dev/null 2>&1; then
  echo "Error: 'go' is required to build opengoal. Install Go 1.21+ first." >&2
  echo "       https://go.dev/dl/" >&2
  exit 1
fi

echo "→ building $BIN"
go build -o "$BIN" ./cmd/og

echo "→ installing $BINDIR/$BIN"
mkdir -p "$BINDIR"
install -m 0755 "$BIN" "$BINDIR/$BIN"

echo "→ installing OpenCode agents into $OPENCODE_DIR/agents/"
mkdir -p "$OPENCODE_DIR/agents"
cp install/agents/*.md "$OPENCODE_DIR/agents/"

echo "→ installing OpenCode slash commands into $OPENCODE_DIR/commands/"
mkdir -p "$OPENCODE_DIR/commands"
cp install/commands/*.md "$OPENCODE_DIR/commands/"

echo
echo "✅ opengoal installed."
echo
case ":$PATH:" in
  *":$BINDIR:"*) ;;
  *)
    echo "⚠️  $BINDIR is not on your PATH. Add this to your shell rc:"
    echo "    export PATH=\"$BINDIR:\$PATH\""
    echo
    ;;
esac

echo "Try it:"
echo "    og help"
echo "    og list-create my-first-roadmap"
echo "    og today"
echo
echo "Inside OpenCode, the slash commands /today, /og, /og-main, /task-add, etc."
echo "are now available. The orchestrator/worker/reviewer agents can be invoked"
echo "with @orchestrator, @worker, @reviewer."
