#!/bin/bash
# Post-build script for zensical/mkdocs site.
# Generates redirect pages and analytics proxy endpoint.

set -euo pipefail

SITE_DIR="${1:?Usage: $0 <site-dir>}"

# --- Redirects ---
# Replaces mkdocs-redirects plugin. Each entry maps an old path (directory URL)
# to a new relative URL.
declare -A REDIRECTS=(
  ["getting_started"]="../getting-started/"
  ["development/controller-design"]="../controller-implementation/"
  ["development/design-decisions"]="../../concepts/design-principles/"
  ["development/coding-convention"]="../coding-standards/"
  ["development/api-contracts"]="../api-design/"
  ["user-guide/drift-detection"]="../../concepts/drift-detection/"
)

# index.md redirects are special: they build to <dir>/index.html directly
declare -A INDEX_REDIRECTS=(
  ["development"]="contributing/"
  ["user-guide"]="../concepts/core-concepts/"
)

for src in "${!REDIRECTS[@]}"; do
  dest="${REDIRECTS[$src]}"
  dir="$SITE_DIR/$src"
  mkdir -p "$dir"
  cat > "$dir/index.html" <<EOF
<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <meta http-equiv="refresh" content="0; url=$dest">
  <link rel="canonical" href="$dest">
</head>
<body>
  <p>This page has moved. If you are not redirected, <a href="$dest">click here</a>.</p>
</body>
</html>
EOF
done

for src in "${!INDEX_REDIRECTS[@]}"; do
  dest="${INDEX_REDIRECTS[$src]}"
  dir="$SITE_DIR/$src"
  mkdir -p "$dir"
  cat > "$dir/index.html" <<EOF
<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <meta http-equiv="refresh" content="0; url=$dest">
  <link rel="canonical" href="$dest">
</head>
<body>
  <p>This page has moved. If you are not redirected, <a href="$dest">click here</a>.</p>
</body>
</html>
EOF
done

# --- Analytics proxy endpoint ---
# Replaces hooks.py on_post_build. Creates an empty script for the Plausible
# analytics reverse-proxy setup.
mkdir -p "$SITE_DIR/t/p"
touch "$SITE_DIR/t/p/script.js"
