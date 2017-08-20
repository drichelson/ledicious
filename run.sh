#!/usr/bin/env bash
set -uxe
LD_LIBRARY_PATH=/home/pi/.gvm/pkgsets/go1.8/global/overlay/lib:
DYLD_LIBRARY_PATH=/home/pi/.gvm/pkgsets/go1.8/global/overlay/lib:
PKG_CONFIG_PATH=/home/pi/.gvm/pkgsets/go1.8/global/overlay/lib/pkgconfig:
[[ -s "/home/pi/.gvm/scripts/gvm" ]] && source "/home/pi/.gvm/scripts/gvm"
#cd ledicious
go env
go build
sudo PORT=80 ./ledicious 2>&1 | logger &