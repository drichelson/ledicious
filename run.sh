#!/usr/bin/env bash
set -uxe
[[ -s "/home/pi/.gvm/scripts/gvm" ]] && source "/home/pi/.gvm/scripts/gvm"
#cd ledicious
go env
go build
sudo PORT=80 ./ledicious 2>&1 | logger &