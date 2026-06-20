#!/bin/bash
# deploy.sh - Build and deploy OneRound server to production
set -e

REMOTE="${DEPLOY_HOST:-debian-01}"
REMOTE_USER="${DEPLOY_USER:-xuanye}"

echo "1. Building server binary (linux/amd64)..."
cd "$(dirname "$0")/../apps/server"
GOOS=linux GOARCH=amd64 go build -o /tmp/oneround-server ./cmd/oneround-server
cd -

echo "2. Uploading to $REMOTE..."
scp /tmp/oneround-server "$REMOTE_USER@$REMOTE:/tmp/oneround-server"

echo "3. Deploying on $REMOTE..."
ssh "$REMOTE_USER@$REMOTE" "
  sudo /usr/bin/systemctl stop oneround
  sudo /usr/bin/install -m 755 -o oneround -g oneround /tmp/oneround-server /opt/oneround/oneround-server
  sudo /usr/bin/systemctl start oneround
  sleep 1
  curl -s http://127.0.0.1:8080/health
"

echo "Done! ✅"
