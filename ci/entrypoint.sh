#!/bin/sh

# cartesi-rollups-node &

# TODO: remove in the future when we stop test the node v2 without espresso in this repo.
# if [ "${CARTESI_FEATURE_AVAIL_READER_ENABLED:-false}" != "false" ]; then
#     cartesi-rollups-avail-reader
# fi

cartesi-rollups-avail-reader

wait
