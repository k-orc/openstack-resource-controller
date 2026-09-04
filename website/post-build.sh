#!/bin/bash
# Post-build script for zensical/mkdocs site.
# Generates the analytics proxy endpoint.
#
# Redirects and HTML/asset minification are now handled natively by
# zensical's built-in `redirects` and `minify` plugin replacements
# (see mkdocs.yml), so this script only needs to cover what zensical
# doesn't support yet: arbitrary MkDocs `hooks`.

set -euo pipefail

SITE_DIR="${1:?Usage: $0 <site-dir>}"

# --- Analytics proxy endpoint ---
# Replaces hooks.py on_post_build. Creates an empty script for the Plausible
# analytics reverse-proxy setup.
mkdir -p "$SITE_DIR/t/p"
touch "$SITE_DIR/t/p/script.js"
