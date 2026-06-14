#!/usr/bin/env bash
# Export project documentation to PDF using pandoc + mermaid-filter.
#
# One-time install:
#   npm i -g mermaid-filter          # Mermaid diagram rendering
#   # pandoc: https://pandoc.org/installing.html
#   # PDF engine (weasyprint): pip install weasyprint
#     OR for a LaTeX-based engine: sudo apt-get install texlive-xetex
#
# Usage: bash scripts/docs-pdf.sh
# PDFs are written alongside each source .md file.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DOCS_DIR="$REPO_ROOT/docs"

# Select PDF engine: prefer weasyprint if available, fall back to default.
if command -v weasyprint >/dev/null 2>&1; then
  ENGINE_ARGS="--pdf-engine=weasyprint"
else
  ENGINE_ARGS=""
fi

find "$DOCS_DIR" -name "*.md" | while read -r f; do
  out="${f%.md}.pdf"
  echo "  $f -> $out"
  pandoc "$f" -F mermaid-filter $ENGINE_ARGS -o "$out"
done

echo "Done."
