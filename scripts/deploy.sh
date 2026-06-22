#!/bin/bash
# deploy.sh - Build and deploy OneRound server to production
set -e

REMOTE="${DEPLOY_HOST:-debian-01}"
DEPLOY_DIR="/opt/oneround"

echo "1. Building server binary (linux/amd64)..."
cd "$(dirname "$0")/../apps/server"
GOOS=linux GOARCH=amd64 go build -o /tmp/oneround-server ./cmd/oneround-server
cd -

echo "2. Uploading to $REMOTE..."
scp /tmp/oneround-server "$REMOTE:/tmp/oneround-server"
scp -r "$(dirname "$0")/../apps/server/migrations" "$REMOTE:$DEPLOY_DIR/"
scp "$(dirname "$0")/../deploy/ecosystem.config.js" "$REMOTE:$DEPLOY_DIR/ecosystem.config.js"

echo "3. Deploying on $REMOTE..."
ssh "$REMOTE" "zsh -i -c \"
  pm2 stop oneround || true
  mkdir -p $DEPLOY_DIR/{database,logs,data}
  mv -f /tmp/oneround-server $DEPLOY_DIR/oneround-server
  cd $DEPLOY_DIR && ./oneround-server -migrate-only -config config.yaml
  cd $DEPLOY_DIR && pm2 startOrRestart ecosystem.config.js
  pm2 save
  sleep 1
  curl -s http://127.0.0.1:8080/health
\""

echo "Done! ✅"
