#!/usr/bin/env sh
# Copyright 2023 Harness, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http:#www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Keeps the govulncheck tool in go.tool.mod on the latest release. Security advisories are only as
# good as the scanner reading them, so we check for a new version before every scan.
#
# Usage:
#   govulncheck-version.sh            compare the pinned version against the latest and offer to upgrade
#   govulncheck-version.sh --update   upgrade to the latest release without asking
#
# Env:
#   GOVULNCHECK_SKIP_CHECK=1    skip the check entirely, e.g. when offline
#   GOVULNCHECK_AUTO_UPDATE=1   upgrade without prompting, e.g. on a machine nobody is watching

set -u

MODFILE="${GOVULNCHECK_MODFILE:-go.tool.mod}"
MODULE="${GOVULNCHECK_MOD:-golang.org/x/vuln}"
PACKAGE="$MODULE/cmd/govulncheck"

pinned_version() {
	go list -m -modfile="$MODFILE" -f '{{.Version}}' "$MODULE" 2>/dev/null
}

upgrade() {
	echo "⬆️  Upgrading govulncheck to the latest release ..."
	go get -tool -modfile="$MODFILE" "$PACKAGE@latest" || return 1
	echo "✅ govulncheck is now $(pinned_version) - remember to commit $MODFILE and ${MODFILE%.mod}.sum"
}

if [ "${1:-}" = "--update" ]; then
	upgrade
	exit $?
fi

if [ -n "${GOVULNCHECK_SKIP_CHECK:-}" ]; then
	echo "⏭  Skipping govulncheck version check (GOVULNCHECK_SKIP_CHECK is set)"
	exit 0
fi

CURRENT=$(pinned_version)
echo "🔍 Checking for a newer govulncheck (pinned: ${CURRENT:-unknown}) ..."

LATEST=$(go list -m -f '{{.Version}}' "$MODULE@latest" 2>/dev/null)
if [ -z "$LATEST" ]; then
	echo "⚠️  Could not reach the module proxy - continuing with ${CURRENT:-the pinned version}"
	exit 0
fi

if [ "$CURRENT" = "$LATEST" ]; then
	echo "✅ govulncheck $CURRENT is up to date"
	exit 0
fi

echo "🆕 govulncheck $LATEST is available (pinned: ${CURRENT:-unknown})"

if [ -n "${GOVULNCHECK_AUTO_UPDATE:-}" ]; then
	upgrade
	exit $?
fi

# No tty means CI, a git hook or a pipe - never block waiting for an answer nobody can give.
if [ ! -t 0 ]; then
	echo "⚠️  Non-interactive shell - staying on ${CURRENT:-the pinned version}."
	echo "   Run 'make govulncheck-update' to upgrade."
	exit 0
fi

printf "   Install %s now? [Y/n] " "$LATEST"
read -r ANSWER
case "$ANSWER" in
	[nN]*) echo "   Staying on ${CURRENT:-the pinned version}" ;;
	*) upgrade || exit 1 ;;
esac