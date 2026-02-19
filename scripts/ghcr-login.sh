#!/usr/bin/env bash
set -euo pipefail

if ! command -v docker >/dev/null 2>&1; then
  echo "docker not found. Install Docker and try again." >&2
  exit 1
fi

read -p "GitHub username: " GH_USERNAME
read -s -p "Enter GHCR PAT: " CR_PAT
echo

echo "Logging in to ghcr.io..."
echo "$CR_PAT" | docker login ghcr.io -u "$GH_USERNAME" --password-stdin

if [ $? -eq 0 ]; then
  echo "Login successful."
  echo "Important: do NOT paste tokens into chat. Revoke exposed tokens immediately." >&2
else
  echo "Login failed." >&2
  exit 1
fi

# Quick verification
if command -v jq >/dev/null 2>&1; then
  echo "Auth entry for ghcr.io:" && jq -r '.auths["ghcr.io"]' ~/.docker/config.json
fi

exit 0
