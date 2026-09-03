#!/bin/zsh
# Dumps the usage views as JSON every few seconds. Any static web server can then show them.
DIR=$(cd "$(dirname "$0")" && pwd)
cd "$DIR/../.."; source "$DIR/demo.env"
C=./bin/colonies; D="$DIR/web/data"; mkdir -p "$D"
dump() { COLONIES_PRVKEY=$1 $C usage $2 --json ${@:3} 2>/dev/null > $D/$3.tmp && mv $D/$3.tmp $D/$4.json; }
while true; do
  COLONIES_PRVKEY=$COLONIES_COLONY_PRVKEY $C usage records --json --count 500 2>/dev/null > $D/records.tmp && mv $D/records.tmp $D/records.json
  COLONIES_PRVKEY=$COLONIES_COLONY_PRVKEY $C usage ls --group-by initiator --json 2>/dev/null > $D/i.tmp && mv $D/i.tmp $D/by_initiator.json
  COLONIES_PRVKEY=$COLONIES_COLONY_PRVKEY $C usage ls --group-by provider  --json 2>/dev/null > $D/p.tmp && mv $D/p.tmp $D/by_provider.json
  COLONIES_PRVKEY=$COLONIES_COLONY_PRVKEY $C usage ls --group-by executor  --json 2>/dev/null > $D/e.tmp && mv $D/e.tmp $D/by_executor.json
  COLONIES_PRVKEY=$COLONIES_COLONY_PRVKEY $C usage balance --json 2>/dev/null > $D/b.tmp && mv $D/b.tmp $D/balance.json
  date -u +%Y-%m-%dT%H:%M:%SZ > $D/updated.txt
  sleep 5
done
