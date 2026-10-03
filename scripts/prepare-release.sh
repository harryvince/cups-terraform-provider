#!/usr/bin/env bash
set -euo pipefail

if [[ -n "$(git status --porcelain)" ]]; then
  echo 'Commit or set aside changes before preparing a release.' >&2
  exit 1
fi
if [[ "$(git branch --show-current)" != main ]]; then
  echo 'Prepare releases from main.' >&2
  exit 1
fi
exec commit-and-tag-version "$@"
