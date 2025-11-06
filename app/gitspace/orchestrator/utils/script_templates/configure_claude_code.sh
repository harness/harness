#!/bin/sh
PROFILE_FILE="/etc/profile"

ANTHROPIC_API_KEY={{ .AnthropicAPIKey }}

# Create ~/.claude directory if it doesn't exist
mkdir -p ~/.claude

# Create settings.json with required content
cat > ~/.claude/settings.json <<EOF
{
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
echo "$ANTHROPIC_API_KEY"
EOF

# Make the get-api-key.sh executable
chmod +x ~/.claude/get-api-key.sh

echo "Successfully added claude settings"