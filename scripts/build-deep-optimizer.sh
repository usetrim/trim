#!/usr/bin/env bash
# Optional: build a frozen Deep Mode binary so users need no pip install.
# Requires Python 3.10+, pip, and several GB of disk/RAM for llmlingua + torch.
# Output lands in dist/trim-deep (or dist/trim-deep.exe on Windows via Git Bash).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT/cli/optimizer"
python3 -m pip install -q "pyinstaller>=6.0" "llmlingua>=0.2.2"
python3 -m PyInstaller --noconfirm --clean trim_deep.spec
echo "Built: $ROOT/cli/optimizer/dist/trim-deep"
echo "Copy next to the trim binary, or set TRIM_OPTIMIZER_BIN to that path."
