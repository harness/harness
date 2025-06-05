#!/bin/bash
set -e

echo "🔍 Checking for nvm..."
export NVM_DIR="$([ -z "${XDG_CONFIG_HOME-}" ] && printf %s "${HOME}/.nvm" || printf %s "${XDG_CONFIG_HOME}/nvm")"

if [ ! -s "$NVM_DIR/nvm.sh" ]; then
  echo "📥 Installing nvm..."
  curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.7/install.sh | bash
fi

# Always source nvm in current shell
echo "🔁 Sourcing nvm..."
[ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"
[ -s "$NVM_DIR/bash_completion" ] && \. "$NVM_DIR/bash_completion"

echo "⬇ Installing Node.js v22 using nvm..."
nvm install 22
nvm use 22
nvm alias default 22

# Update PATH
export PATH="$NVM_DIR/versions/node/v22.*/bin:$PATH"

echo "📦 Installing @openai/codex CLI..."
npm install -g @openai/codex

echo "✅ Codex installed. Verifying..."
codex --version

# Set up workspace
WORKDIR="$HOME/home/hackweek/gobeyond"
cd "$WORKDIR"

# Save current Git state
echo "🔧 Stashing current changes if any..."
git diff > "$WORKDIR/codex-pre.patch" || true

# Path to the lint log file
LINT_LOG="$WORKDIR/lint-output.txt"
LINT_CONTENT=$(cat "$LINT_LOG")

echo "📄 Preparing Codex prompt..."
#PROMPT_FILE="$WORKDIR/codex_prompt.txt"

PROMPT_FILE="
You are a strict, deterministic Go linter fixer. You will directly apply changes to Go source files based on the following golangci-lint log.

## Objective:
Apply lint fixes to the actual Go codebase files, not as suggestions or examples. All changes must be made directly in-place, modifying the original files exactly where needed.

## Constraints:
- Only modify **exact files and lines** referenced in the lint log.
- Apply the **minimum code change** necessary to resolve each issue.
- **Do not** suggest fixes — directly apply them to the code.
- Do not change code outside of lines explicitly required to fix.
- Do not reformat imports unless required by a specific lint rule.
- Avoid structural rewrites unless demanded by the lint warning.
- Adhere strictly to idiomatic Go best practices and formatting.
- Do not include explanations or surrounding context — only edit the code.
- Edits must be deterministic and reproducible.
- Do not skip any errors unless marked to ignore.

---

## 🔧 Lint log input:
$LINT_CONTENT

Now modify the source files accordingly and apply the fixes in place.
"

echo "🚀 Running Codex to fix lint issues..."
codex -q -a full-auto --model gpt-4.1 --fullAutoErrorMode ignore-and-continue --workdir "$WORKDIR" "$PROMPT_FILE"

echo "🎨 Formatting Go code after changes..."
go fmt ./...

echo "🔍 Capturing git diff..."
git diff > "$WORKDIR/codex-diff.patch"

echo "✅ Done. Changes saved in: $WORKDIR/codex-diff.patch"
