#!/usr/bin/env bash

set -euo pipefail

workdir=$(pwd)
. scripts/vars.sh

(cd "${workdir}" && scripts/download-vd.sh)
(cd "${workdir}" && scripts/bootstrap-vm.exp)
(cd "${workdir}" && scripts/setup-vm.sh)

rsync -rltpDP -e "ssh ${SSH_OPTS[*]}" --exclude "*.qcow2*" "${workdir}/" root@localhost:/root/runner
"${SSH[@]}" "sh -c 'cd /root/runner && scripts/compile.sh'"
rsync -rltpDP -e "ssh ${SSH_OPTS[*]}" --delete root@localhost:/root/runner/build/. build/