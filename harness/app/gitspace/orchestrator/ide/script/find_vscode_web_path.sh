#!/bin/sh

echo "START_MARKER"$(find / -type d -path '*code-server/*src/browser/media*' -print -quit 2>/dev/null)"END_MARKER"
