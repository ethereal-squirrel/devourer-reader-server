#!/usr/bin/env bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SERVER_DIR="$(dirname "$SCRIPT_DIR")"
CLIENT_DIR="$SERVER_DIR/../client"
CLIENT_DIST="$CLIENT_DIR/dist"
SERVER_CLIENT="$SERVER_DIR/client"

echo "Building client..."
(cd "$CLIENT_DIR" && npm run build)

echo "Removing old client from server..."
rm -rf "$SERVER_CLIENT"

echo "Copying dist to server/client..."
cp -r "$CLIENT_DIST" "$SERVER_CLIENT"

echo "Done."
