set -x

export CACHE_PATH=opensense-tools-cache.tar.gz
export HOST=root@localhost

SSH_OPTS=(-i ~/.ssh/id_rsa)
SSH_OPTS+=(-p 2222)
SSH_OPTS+=(-o StrictHostKeyChecking=no)
SSH_OPTS+=(-o UserKnownHostsFile=/dev/null)
export SSH_OPTS

SSH=(ssh)
SSH+=(${SSH_OPTS[@]})
SSH+=("$HOST")
export SSH

set +x