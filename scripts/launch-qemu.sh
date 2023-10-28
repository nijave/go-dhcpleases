#!/bin/sh

set -u

vd=$(find . -name "*.qcow2" | head -n 1)

case $(uname -s) in
    Linux)
        accel=,accel=kvm
        ;;
    Darwin)
        accel=,accel=hvf
	;;
    *)
        accel=""
esac

if pkill -f "$vd"; then
	printf "waiting on previous qemu to exit"
    i=0
    while pgrep -f "$vd" >/dev/null; do
        printf " %d" "$i"
	    sleep 0.5
        i+=1
    done
    echo ""
fi

qemu-system-x86_64 \
    -machine type=q35${accel} \
    -smp sockets=1,cores=3 \
    -m 6G \
    -drive "format=qcow2,file=$vd,id=drive0,if=virtio" \
    -device virtio-scsi-pci,id=drive0 \
    -device e1000,netdev=net0 \
    -netdev user,id=net0,hostfwd=tcp:127.0.0.1:2222-:22 \
    -drive if=pflash,format=raw,readonly=on,file=$(dirname "$0")/OVMF_CODE.fd \
    -cpu host,-pdpe1gb \
    -serial mon:stdio \
    -nographic
