#!/usr/bin/env bash
set -ue
#cd ..
ssh pi@pi.local sudo pkill ledicious || echo "killed"
ssh pi@pi.local rm -rf ledicious/*
rsync -avz -e "ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null" . pi@pi.local:ledicious/
#ssh pi@pi.local "cd ledicious && go env && go build"
#ssh pi@pi.local "cd ledicious && ./run.sh"
