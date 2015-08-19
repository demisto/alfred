#!/bin/bash
set -m

freshclam -d &
/worker -conf /etc/g-worker.conf -loglevel=debug &

pids=`jobs -p`

exitcode=0

function terminate() {
    trap "" CHLD

    for pid in $pids; do
        if ! kill -0 $pid 2>/dev/null; then
            wait $pid
            exitcode=$?
        fi
    done

    kill $pids 2>/dev/null
}

trap terminate CHLD
wait

exit $exitcode
