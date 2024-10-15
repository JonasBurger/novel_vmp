#!/usr/bin/env bash
set -ue

# Change to the directory of the script
cd "$(dirname "$0")"

# Add the upstream remote if it doesn't exist
git remote add upstream https://github.com/deviantony/docker-elk || true

# update repo
git fetch upstream
git checkout main
git merge upstream/main --no-edit
git push