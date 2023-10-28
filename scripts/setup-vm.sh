#!/usr/bin/env bash

set -euo pipefail

. scripts/vars.sh

printf "waiting for vm ssh"
for i in {1..60}; do
  sshpass -p root ssh "${SSH_OPTS[@]}" "$HOST" exit 0 2>/dev/null && break
  printf " %d" "$i"
  sleep 2
done
echo ""

set -x

sshpass -p root ssh-copy-id "${SSH_OPTS[@]}" "$HOST"

"${SSH[@]}" "pkg install -y git lang/go pigz rsync zstd php83 python311"

if [ -f "$CACHE_PATH" ]; then
    scp "${SSH_OPTS[@]}" "$CACHE_PATH" "${HOST}:~/"
fi

"${SSH[@]}" <<- EOF
    set -x
    
    # opnsense build scripts look for "python3" so link newest versioned binary
    newest_python=$(find /usr/local/bin | rev | cut -d/ -f 1 | rev  | grep -E 'python3\.[0-9]+$' | sort -k 2 -t . -n -r | head -n 1)
    ln -s "$(which $newest_python)" /usr/local/bin/python3

    if [ -f "~/$CACHE_PATH" ]; then
        tar -C / -xf "~/$CACHE_PATH"
    else
        git clone https://github.com/opnsense/tools /usr/tools
    fi

    cd /usr/tools && make update
EOF

"${SSH[@]}" <<EOF
time tar -cf - /usr/tools /usr/core /usr/plugins /usr/ports /usr/src | pigz > ~/$CACHE_PATH
EOF

scp "${SSH_OPTS[@]}" "${HOST}:~/$CACHE_PATH" "$CACHE_PATH"