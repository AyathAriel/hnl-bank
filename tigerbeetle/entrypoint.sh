#!/bin/sh
set -e

DATA_FILE="/data/0_0.tigerbeetle"

if [ ! -f "$DATA_FILE" ]; then
    echo "tigerbeetle: formatting new data file at $DATA_FILE"
    /tigerbeetle format \
        --cluster="${TB_CLUSTER_ID:-0}" \
        --replica=0 \
        --replica-count=1 \
        --development \
        "$DATA_FILE"
else
    echo "tigerbeetle: data file already exists, skipping format"
fi

echo "tigerbeetle: starting on 0.0.0.0:3000"
exec /tigerbeetle start --addresses=0.0.0.0:3000 --development "$DATA_FILE"
