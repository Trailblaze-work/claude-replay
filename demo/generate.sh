#!/usr/bin/env bash
#
# Regenerate the demo GIF from a Claude Code session.
#
# Usage:
#   ./demo/generate.sh                  # use default session
#   ./demo/generate.sh <session-id>     # use a specific session
#
# Requirements:
#   - agg (cargo install agg)
#   - go (to build claude-replay)
#
set -euo pipefail
cd "$(dirname "$0")/.."

SESSION="${1:-76e0177c}"
OUTPUT="demo.gif"

echo "Building claude-replay..."
go build -o /tmp/claude-replay .

echo "Exporting session ${SESSION} as GIF..."
/tmp/claude-replay export "$SESSION" \
    --format gif \
    --mode compressed \
    --width 120 \
    --height 40 \
    -o "$OUTPUT"

echo ""
echo "Done: $OUTPUT ($(du -h "$OUTPUT" | cut -f1 | xargs))"
