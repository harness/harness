#!/bin/sh
PROFILE_FILE="/etc/profile"

ANTHROPIC_API_KEY={{ .AnthropicAPIKey }}
ANTHROPIC_AUTH_TOKEN={{ .AuthToken }}

# Create ~/.claude directory if it doesn't exist
mkdir -p ~/.claude

# Create settings.json with required content
cat > ~/.claude/settings.json <<EOF
{
{{- if .BaseURL }}
  "env": {
    "ANTHROPIC_BASE_URL": "{{ .BaseURL }}",
    "ANTHROPIC_MODEL": "{{ .Model }}",
    "ANTHROPIC_SMALL_FAST_MODEL": "{{ .Model }}",
    "ANTHROPIC_API_KEY": ""
  },
{{- end }}
  "apiKeyHelper": "~/.claude/get-api-key.sh",
  "permissions": {
      "allow": [
        "Read(**)",
        "Edit(**)",
        "Bash(ls:*)",
        "Bash(grep:*)",
        "Bash(git status:*)",
        "Bash(git diff:*)",
        "Bash(git add:*)",
        "Bash(git commit:*)",
        "Bash(npm test:*)",
        "Bash(yarn test:*)"
      ],
      "deny": []
    }
}
EOF

# Create get-api-key.sh with required content
cat > ~/.claude/get-api-key.sh <<EOF
#!/bin/sh
{{- if .BaseURL }}
echo "$ANTHROPIC_AUTH_TOKEN"
{{- else }}
echo "$ANTHROPIC_API_KEY"
{{- end }}
EOF

# Make the get-api-key.sh executable
chmod +x ~/.claude/get-api-key.sh

echo "Successfully added claude settings"
