#!/bin/zsh
# Submits a random job as a random user every 15-30 seconds.
DIR=$(cd "$(dirname "$0")" && pwd)
cd "$DIR/../.."; source "$DIR/demo.env"; C=./bin/colonies
users=(alice:$ALICE_KEY bob:$BOB_KEY carol:$CAROL_KEY)
cpujobs=(train:4000m preprocess:2000m build:1000m simulate:8000m)
gpujobs=(infer:1 finetune:2 render:1)
while true; do
  u=${users[$((RANDOM % 3 + 1))]}; name=${u%%:*}; key=${u##*:}
  if (( RANDOM % 2 )); then j=${cpujobs[$((RANDOM % 4 + 1))]}; f=${j%%:*}; cpu=${j##*:}; type=cpu; gpu=0
  else j=${gpujobs[$((RANDOM % 3 + 1))]}; f=${j%%:*}; cpu=2000m; type=gpu; gpu=${j##*:}; fi
  printf '{"funcname":"%s","conditions":{"colonyname":"dev","executortype":"%s","cpu":"%s","mem":"4Gi","gpu":{"name":"a100","count":%s}},"maxexectime":300,"maxretries":0,"label":"%s-%s"}' $f $type $cpu $gpu $name $f > "$DIR/gen.json"
  $C function submit --spec "$DIR/gen.json" --prvkey $key >/dev/null 2>&1
  sleep $(( 15 + RANDOM % 16 ))
done
