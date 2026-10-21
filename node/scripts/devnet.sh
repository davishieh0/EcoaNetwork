#!/usr/bin/env bash
# Local devnet in tmux: N nodes, each bootstrapping to the ones before it.
# Node 1 writes to data/bbolt/bolt.db, node N to data/bbolt/bolt-N.db; both
# survive restarts.
#
#   scripts/devnet.sh [N]       start N nodes (default 3) and attach
#   scripts/devnet.sh publish   publish a fresh event from a throwaway node
set -euo pipefail

ROOT=$(cd "$(dirname "$0")/.." && pwd)
SELF=$ROOT/scripts/devnet.sh
BIN=$ROOT/bin/ecoa
DIR=${TMPDIR:-/tmp}/ecoa-devnet
SESSION=ecoa-devnet


node() {
	local i=$1
	local boot
	boot=$(paste -sd, "$DIR/bootstrap")

	# Node 1 shares bolt.db with make run; the rest get their own file.
	local db=$ROOT/data/bbolt/bolt.db
	if ((i > 1)); then
		db=$ROOT/data/bbolt/bolt-$i.db
	fi

	BOLT_DB_PATH=$db "$BIN" \
		-listen "/ip4/127.0.0.1/tcp/$((4000 + i))" \
		-grpc "127.0.0.1:$((5000 + i))" \
		-bootstrap "$boot" 2>&1 | tee "$DIR/node$i.log" || true

	echo "node $i exited; press Enter to close"
	read -r
}


publish() {
	local id
	id="evt_$(date +%s%N)"
	printf '{"eventId":"%s","eventType":"review.created"}\n' "$id" \
		> "$DIR/$id.json"

	echo "publishing $id"
	BOLT_DB_PATH=$DIR/$id.db timeout -s INT 5 "$BIN" \
		-listen /ip4/127.0.0.1/tcp/0 \
		-bootstrap "$(paste -sd, "$DIR/bootstrap")" \
		-publish "$DIR/$id.json" > "$DIR/$id.log" 2>&1 || true
	echo "done; each node pane should log \"stored\""
}


wait_addr() {
	local i=$1
	local addr

	for _ in $(seq 50); do
		addr=$(grep -o "/ip4/127.0.0.1/tcp/$((4000 + i))/p2p/[A-Za-z0-9]*" \
			"$DIR/node$i.log" 2>/dev/null | head -1 || true)

		if [[ -n $addr ]]; then
			echo "$addr" >> "$DIR/bootstrap"
			return
		fi

		sleep 0.2
	done

	echo "node $i did not start; see $DIR/node$i.log" >&2
	exit 1
}


start() {
	local n=${1:-3}

	make -C "$ROOT" build > /dev/null
	tmux kill-session -t "$SESSION" 2> /dev/null || true
	rm -rf "$DIR"
	mkdir -p "$DIR"
	: > "$DIR/bootstrap"

	for ((i = 1; i <= n; i++)); do
		if ((i == 1)); then
			tmux new-session -d -s "$SESSION" -x 200 -y 50 \
				"bash '$SELF' node $i"
		else
			tmux split-window -t "$SESSION" "bash '$SELF' node $i"
			tmux select-layout -t "$SESSION" tiled > /dev/null
		fi

		wait_addr "$i"
	done

	tmux split-window -t "$SESSION"
	tmux select-layout -t "$SESSION" tiled > /dev/null
	tmux send-keys -t "$SESSION" "bash '$SELF' publish"

	if [[ -n ${TMUX:-} ]]; then
		tmux switch-client -t "$SESSION"
	else
		tmux attach -t "$SESSION"
	fi
}


case ${1:-} in
	node)    node "$2" ;;
	publish) publish ;;
	*)       start "${1:-3}" ;;
esac
