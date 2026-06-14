#!/usr/bin/env bash
# Export project documentation to PDF using pandoc + mermaid-filter (Mermaid preserved).
#
# One-time install (no sudo: Homebrew + bun):
#   brew install pandoc weasyprint
#   bun add -g mermaid-filter            # or: npm i -g mermaid-filter
#   # A Chromium/Chrome must be present (mermaid-filter renders diagrams headless).
#
# Usage (as your normal user — NOT sudo, NOT sh):
#   bash scripts/docs-pdf.sh
# sudo/sh lose your PATH and pandoc/mermaid-filter (user installs) won't be found.
# PDFs are written alongside each source .md (build artifacts; gitignored).
set -euo pipefail

# Preflight: helpful errors instead of a cryptic "command not found".
if [ "$(id -u)" = "0" ]; then
  echo "ERROR: run as your normal user, NOT sudo (root lacks your brew/bun PATH)." >&2
  exit 1
fi
for tool in pandoc mermaid-filter; do
  command -v "$tool" >/dev/null 2>&1 || {
    echo "ERROR: '$tool' not on PATH. Install: brew install pandoc weasyprint && bun add -g mermaid-filter" >&2
    echo "       Then run with bash (not sh): bash scripts/docs-pdf.sh" >&2
    exit 1
  }
done

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DOCS_DIR="$REPO_ROOT/docs"
cd "$REPO_ROOT"

# PDF engine: prefer weasyprint, else pandoc default (LaTeX).
ENGINE_ARGS=""
command -v weasyprint >/dev/null 2>&1 && ENGINE_ARGS="--pdf-engine=weasyprint"

# Code-block syntax theme. Dracula (dark block, high contrast) by default.
# Swap to a built-in if preferred: tango | kate | breezedark | zenburn | espresso.
HIGHLIGHT_STYLE="${HIGHLIGHT_STYLE:-$REPO_ROOT/scripts/dracula.theme}"

# Render Mermaid diagrams as vector SVG (crisp/zoomable) instead of raster PNG.
export MERMAID_FILTER_FORMAT="${MERMAID_FILTER_FORMAT:-svg}"

# htmlLabels:false -> Mermaid draws labels as native SVG <text> instead of
# <foreignObject> (HTML), which weasyprint cannot render (text vanishes otherwise).
MMD_CFG="$REPO_ROOT/.mermaid-config.json"
printf '{ "htmlLabels": false, "flowchart": { "htmlLabels": false } }\n' >"$MMD_CFG"

# mermaid-filter needs Chromium; point it at the system browser with --no-sandbox.
CHROME="${PUPPETEER_EXECUTABLE_PATH:-}"
for c in chromium-browser chromium google-chrome google-chrome-stable; do
  [ -n "$CHROME" ] && break
  command -v "$c" >/dev/null 2>&1 && CHROME="$(command -v "$c")"
done
PUP_CFG="$REPO_ROOT/.puppeteer.json"
[ -n "$CHROME" ] && printf '{ "executablePath": "%s", "args": ["--no-sandbox", "--disable-gpu", "--disable-dev-shm-usage"] }\n' "$CHROME" >"$PUP_CFG"
cleanup() { rm -f "$PUP_CFG" "$MMD_CFG" "$REPO_ROOT/mermaid-filter.err"; }
trap cleanup EXIT

# -f gfm: GitHub-flavored anchors so the in-doc índice links resolve in the PDF.
find "$DOCS_DIR" -name "*.md" | sort | while read -r f; do
  out="${f%.md}.pdf"
  echo "  $f -> $out"
  pandoc -f gfm "$f" -F mermaid-filter $ENGINE_ARGS --highlight-style="$HIGHLIGHT_STYLE" -o "$out"
done

echo "Done."
