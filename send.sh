#!/usr/bin/env bash
set -ue
HOSTNAME=globe.local

#cd ..
ssh pi@${HOSTNAME} sudo pkill ledicious || echo "killed"
ssh pi@${HOSTNAME} rm -rf ledicious/*
rsync -avz -e "ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null" . pi@${HOSTNAME}:ledicious/
#ssh pi@pi.local "cd ledicious && go env && go build"
ssh pi@${HOSTNAME} "cd ledicious && ./run.sh" &
