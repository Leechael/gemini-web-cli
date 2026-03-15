#!/usr/bin/env bash
set -euo pipefail

# Usage: ./scripts/next-version.sh <patch|minor|major> [alpha|beta|rc]
# Output: new version without prefix (e.g. 1.2.4 or 1.3.0-beta.1)

BUMP="${1:-patch}"
PRE="${2:-}"

case "$BUMP" in
  patch|minor|major) ;;
  *)
    echo "invalid bump: $BUMP" >&2
    exit 1
    ;;
esac

if [[ -n "$PRE" ]]; then
  case "$PRE" in
    alpha|beta|rc) ;;
    *)
      echo "invalid prerelease tag: $PRE" >&2
      exit 1
      ;;
  esac
fi

env_file="release-naming.env"
if [[ ! -f "$env_file" ]]; then
  echo "missing naming contract file: $env_file" >&2
  exit 1
fi

# shellcheck disable=SC1090
source "$env_file"

if [[ -z "${TAG_PREFIX:-}" ]]; then
  echo "TAG_PREFIX is empty in $env_file" >&2
  exit 1
fi

latest_tag=$(git tag -l "${TAG_PREFIX}*" --sort=-version:refname | head -n1)

if [[ -z "$latest_tag" ]]; then
  base_version="0.0.0"
else
  if [[ "$latest_tag" == ${TAG_PREFIX}* ]]; then
    base_version="${latest_tag#${TAG_PREFIX}}"
  else
    base_version="0.0.0"
  fi
  base_version="${base_version%%-*}"
fi

IFS='.' read -r major minor patch <<< "$base_version"
major=${major:-0}
minor=${minor:-0}
patch=${patch:-0}

case "$BUMP" in
  patch)
    patch=$((patch + 1))
    ;;
  minor)
    minor=$((minor + 1))
    patch=0
    ;;
  major)
    major=$((major + 1))
    minor=0
    patch=0
    ;;
esac

version="${major}.${minor}.${patch}"
if [[ -n "$PRE" ]]; then
  version="${version}-${PRE}.1"
fi

echo "$version"
