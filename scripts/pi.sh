#!/bin/bash

function log() {
    echo "$(date +"%F %T.%3N"): $*"
}

function download_photos() {
    dest=$1
    rm -rf ${dest}
    mkdir -p ${dest}

    export OUTPUT_DIR=${dest}
    export CONFIG=/home/pi/gphotos-sync-secret.json
    export NPHOTOS=150

    log "Updating photos"
    cd /home/pi/rand-gphotos
    ./rand-gphotos-pi
}

function copy_photos() {
    img_source=$1
    img_dest=$2
    fat_file=$3

    log "Umounting old FS"
    sudo umount ${img_dest} || true
    rm -rf ${img_dest}
    sudo modprobe -r g_mass_storage
    rm -f ${fat_file}
    log "Creating new FS"
    mkdir -p $(dirname ${fat_file})
    dd bs=1M if=/dev/zero of=${fat_file} count=16
    /usr/sbin/mkdosfs ${fat_file} -F 16 -I
    mkdir -p ${img_dest}
    sudo mount -ousers,umask=000 ${fat_file} ${img_dest}
    cp ${img_source}/*.jpg ${img_dest}/
    sudo sync
    sudo umount ${img_dest}
    load_fs ${fat_file}
}

function load_fs() {
    fat_file=$1
    log "Loading mass storage device"
    sudo modprobe g_mass_storage file=${fat_file} stall=0 ro=0 removable=0 idVendor=${RANDOM} idProduct=${RANDOM} iSerialNumber=${RANDOM}
    log "Loaded mass storage device"
}

function init_mass_storage() {
    img_src="/home/pi/mass_storage/tmp-download"
    img_dest="/home/pi/mass_storage/usb_share"
    fat_file="/home/pi/mass_storage/piusb.bin"
    log_file="/home/pi/mass_storage/gphotos.log"
    arg="$1"

    exec >>${log_file}
    exec 2>&1

    log "Script init"

    if [[ "$arg" == "loadonly" ]]; then
        if [ -f ${fat_file} ]; then
            load_fs ${fat_file}
        fi
        exit 0
    fi

    download_photos ${img_src}
    copy_photos ${img_src} ${img_dest} ${fat_file}
}

set -eo pipefail
sudo touch /var/lock/init_mass_storage
sudo chmod 666 /var/lock/init_mass_storage
(
    set -eo pipefail
    if ! flock -n 9; then
        echo "lock not available"
        exit 1
    fi
    init_mass_storage $1
) 9>/var/lock/init_mass_storage
