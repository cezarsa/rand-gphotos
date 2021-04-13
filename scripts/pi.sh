#!/bin/bash

set -eo pipefail

function init_mass_storage() {
    if ! [ -f /tmp/piusb.bin ]; then
        echo "Initializing vfat FS"
        dd bs=1M if=/dev/zero of=/tmp/piusb.bin count=64
        mkdosfs /tmp/piusb.bin -F 32 -I
        mkdir -p /tmp/usb_share
        sudo mount -ousers,umask=000 /tmp/piusb.bin /tmp/usb_share
        cp /home/pi/rand-gphotos/placeholders/*.jpg /tmp/usb_share/
        sync
        sudo modprobe g_mass_storage file=/tmp/piusb.bin stall=0 ro=1
    fi

    sudo rm -rf /tmp/img-download
    mkdir -p /tmp/img-download

    export OUTPUT_DIR=/tmp/img-download
    export CONFIG=/home/pi/gphotos-sync-secret.json

    echo "Updating photos"
    cd /home/pi/rand-gphotos
    ./rand-gphotos-pi

    echo "Reloading mass storage"
    sudo modprobe -r g_mass_storage
    sleep 1
    cp --no-preserve=mode,ownership /tmp/img-download/*.jpg /tmp/usb_share/
    sync
    sudo modprobe g_mass_storage file=/tmp/piusb.bin stall=0 ro=1
    sudo rm -rf /tmp/img-download
}

(
    if ! flock -n 9; then
        echo "lock not available"
        exit 1
    fi
    init_mass_storage
) 9>/var/lock/init_mass_storage
