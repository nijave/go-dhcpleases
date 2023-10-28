#!/usr/bin/env bash

set -x

release=13.2-RELEASE

disk=FreeBSD-${release}-amd64.qcow2.xz
ext_disk=$(basename -s .xz "$disk")
url=https://download.freebsd.org/releases/VM-IMAGES/${release}/amd64/Latest/${disk}

uname -s

rm -f *.qcow2

# Check for decompressed, unmodified copy
if [ ! -f "${ext_disk}.bak" ]; then
    if [ ! -f "$disk" ]; then
        time wget "$url"
    fi
    time xz -d "$disk"
    mv "${ext_disk}" "${ext_disk}.bak"
fi

# Much faster to copy on repeat reruns than decompress
cp "${ext_disk}.bak" "${ext_disk}"

qemu-img resize "$ext_disk" 20G
