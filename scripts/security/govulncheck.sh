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

# Runs govulncheck and fails only on vulnerabilities we can actually act on, i.e. the ones with an
# upstream fix. Vulnerabilities without a fixed version are still printed in full, but they don't
# break the build - there is nothing to upgrade to yet.
#
# Every argument is forwarded to govulncheck, so `govulncheck.sh ./...` scans the whole module.

set -u

GOVULNCHECK="${GOVULNCHECK:-go tool -modfile=go.tool.mod govulncheck}"

REPORT=$(mktemp)
# shellcheck disable=SC2064 # expand REPORT now, it is not going to change
trap "rm -f $REPORT" EXIT

# govulncheck writes the report to stdout and its progress to stderr, so only stdout is captured.
# shellcheck disable=SC2086 # GOVULNCHECK is a command line and has to be split into words
$GOVULNCHECK "$@" > "$REPORT"
STATUS=$?

cat "$REPORT"

# 0 means nothing was found and 3 means vulnerabilities were found in code that is actually
# reachable. Every other exit code is govulncheck itself failing, which we always propagate.
if [ "$STATUS" -eq 0 ]; then
	exit 0
fi
if [ "$STATUS" -ne 3 ]; then
	echo "❌ govulncheck failed with exit code $STATUS" >&2
	exit "$STATUS"
fi

FIXABLE=$(awk '/^[[:space:]]*Fixed in:/ && $3 != "N/A" { n++ } END { print n+0 }' "$REPORT")
UNFIXABLE=$(awk '/^[[:space:]]*Fixed in:/ && $3 == "N/A" { n++ } END { print n+0 }' "$REPORT")

# Fail closed. If the report can't be parsed we must not claim the code is clean, so treat
# everything as actionable and let a human look at the output above.
if [ "$FIXABLE" -eq 0 ] && [ "$UNFIXABLE" -eq 0 ]; then
	echo "❌ govulncheck reported vulnerabilities but no 'Fixed in:' line could be parsed" >&2
	echo "   Treating them as actionable - please review the report above." >&2
	exit "$STATUS"
fi

echo ""
if [ "$FIXABLE" -gt 0 ]; then
	echo "❌ $FIXABLE vulnerability(s) with a fix available:"
	awk '
		/^[[:space:]]*Vulnerability #/ { id = $3 }
		/^[[:space:]]*Found in:/       { found = $3 }
		/^[[:space:]]*Fixed in:/ && $3 != "N/A" { printf "   • %s: %s -> %s\n", id, found, $3 }
	' "$REPORT"
	if [ "$UNFIXABLE" -gt 0 ]; then
		echo "ℹ️  $UNFIXABLE more vulnerability(s) have no upstream fix yet - reported only."
	fi
	exit 1
fi

echo "⚠️  $UNFIXABLE vulnerability(s) found, none of them fixed upstream yet - reported only."
echo "   Nothing to upgrade to, so this is not a failure. Keep an eye on the advisories above."
exit 0