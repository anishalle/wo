#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

PREFIX="${PREFIX:-$HOME/.local}"
BINDIR="${BINDIR:-$PREFIX/bin}"
MANDIR="${MANDIR:-$PREFIX/share/man/man1}"

make BINDIR="$BINDIR" MANDIR="$MANDIR" install

echo "Installed wo to $BINDIR/wo"
echo ""
echo "Add shell integration:"
echo "  zsh:  eval \"$(wo init zsh)\""
echo "  bash: eval \"$(wo init bash)\""
echo "  fish: wo init fish | source"
