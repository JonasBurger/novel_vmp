#!/usr/bin/env bash
docker ps -a --format '{{.ID}} {{.Image}}' | grep 'scanner_' | awk '{print $1}' | xargs -I {} sh -c 'docker stop {}; docker rm {}'
docker ps -a --format '{{.ID}} {{.Image}}' | grep 'python \./main\.py' | awk '{print $1}' | xargs -I {} sh -c 'docker stop {}; docker rm {}'

