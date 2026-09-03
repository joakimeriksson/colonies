# Usage tracing demo

A self-running colony with three users and two fake executors, and a live web
page over the usage ledger. Nothing here changes the server; the page is fed by
`colonies usage ... --json` written to files every five seconds.

## Setup

1. Start a server from the `usage-tracing-poc` branch and create the colony,
   three users and two executors. `PLAN.md` has the exact commands. Give each
   executor an owner: `colonies executor add ... --owner alice --approve`.
2. `cp demo.env.example demo.env` and fill in the keys.

## Run

```bash
./fake_executor.sh $ABOX_KEY 10 40 &   # polls, sleeps 10-40 s per job, closes it
./fake_executor.sh $BGPU_KEY 20 60 &
./job_generator.sh &                   # a random job from a random user every 15-30 s
./refresh.sh &                         # dumps usage JSON into web/data
(cd web && python3 -m http.server 8099 --bind 127.0.0.1)
```

Open http://127.0.0.1:8099. The viewer buttons show what each identity may see.

The fake executors must keep polling: the server unregisters an executor it has
not heard from for ten minutes (`COLONIES_STALE_EXECUTOR_DURATION`). Re-approve
with `colonies executor approve --name <name>` if that happens.

## CLI equivalents

```bash
colonies usage ls --group-by initiator|provider|executor|label
colonies usage ls --provider me --group-by initiator
colonies usage balance
colonies usage records
```
