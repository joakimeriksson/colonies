#!/bin/zsh
# A fake executor: polls for work, "runs" it by sleeping, closes it. One in six jobs fails.
# usage: fake_executor.sh <executor key> <min secs> <max secs>
DIR=$(cd "$(dirname "$0")" && pwd)
cd "$DIR/../.."; source "$DIR/demo.env"; C=./bin/colonies
KEY=$1; MIN=${2:-10}; MAX=${3:-40}
while true; do
  P=$($C process assign --prvkey $KEY --timeout 20 2>&1 | grep -o 'ProcessId=[0-9a-f]*' | head -1 | cut -d= -f2)
  [[ -z "$P" ]] && continue
  sleep $(( MIN + RANDOM % (MAX - MIN + 1) ))
  if (( RANDOM % 6 == 0 )); then $C process fail -p $P --errors "simulated failure" --prvkey $KEY >/dev/null 2>&1
  else $C process close -p $P --prvkey $KEY >/dev/null 2>&1; fi
done
