#!/bin/bash

set -eo pipefail

rm -rf /tmp/img-download
mkdir -p /tmp/img-download

export OUTPUT_DIR=/tmp/img-download
export CONFIG=/home/pi/gphotos-sync-secret.json

cd /home/pi/rand-gphotos
./rand-gphotos-pi

sudo modprobe -r g_mass_storage
sleep 1
cp --no-preserve=mode,ownership /tmp/img-download/*.jpg /mnt/usb_share/
sync
sudo modprobe g_mass_storage file=/piusb.bin stall=0 ro=1