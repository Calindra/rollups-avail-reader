#!/bin/bash
set -e

rm -rf ./rollups-node

git clone -b v2.0.0-alpha.1 --depth 1 --recurse-submodules https://github.com/cartesi/rollups-node.git

docker stop $(docker ps -q) || true

docker buildx prune --all --force && docker system prune --volumes --force

docker run -d --rm --name postgres -p 5432:5432 -e POSTGRES_PASSWORD=password -e POSTGRES_DB=rollupsdb postgres:16-alpine

echo "Migrate DB node v2"
cd rollups-node
eval $(make env)
export CGO_CFLAGS="-D_GNU_SOURCE -D__USE_MISC"
go run dev/migrate/main.go
cd -

echo "Migrate DB Espresso"
eval $(make env)
make migrate
make generate-db

echo "Build image"
#docker build -t espresso .
docker build -t espresso -f ./ci/Dockerfile .

echo "Run Anvil"
cd rollups-node
make devnet
make start-devnet
cd -

# export $(grep -v '^#' env.nodev2-sepolia | xargs)
export CARTESI_BLOCKCHAIN_HTTP_ENDPOINT=https://eth-sepolia.g.alchemy.com/v2/9hjbdwjACHkHf1j01yE2j7Q9G9J1VsC9
export CARTESI_BLOCKCHAIN_WS_ENDPOINT=wss://eth-sepolia.g.alchemy.com/v2/9hjbdwjACHkHf1j01yE2j7Q9G9J1VsC9

# go test --timeout 1m -p 1 ./...
# export AVAIL_STARTING_BLOCK=$(curl -s https://query.decaf.testnet.espresso.network/v0/status/block-height)

docker run --env-file ./ci/env.nodev2-local --rm --network=host --name c_espresso espresso
# docker run --env-file ./ci/env.nodev2-sepolia \
#   -e CARTESI_BLOCKCHAIN_HTTP_ENDPOINT=$CARTESI_BLOCKCHAIN_HTTP_ENDPOINT \
#   -e CARTESI_BLOCKCHAIN_WS_ENDPOINT=$CARTESI_BLOCKCHAIN_WS_ENDPOINT \
#   --rm --network=host --name c_espresso espresso

exit 0

# local anvil deploy
docker exec c_espresso cartesi-rollups-cli app deploy -n echo-dapp -t applications/echo-dapp/ -v

# sepolia deploy
export ACCOUNT=0xB5C1674c0527b6C31A5019fD04a6C1529396DA37
export PRIVATE_KEY=ad03...e462
export RPC_URL=https://eth-sepolia.g.alchemy.com/v2/<key>
docker exec c_espresso cartesi-rollups-cli app deploy -v -n echo-dapp -o $ACCOUNT -O $ACCOUNT -t applications/echo-dapp/

# Milton
docker exec c_espresso cartesi-rollups-cli app register -v -c 0xe82D9ebc0c2773516914a1285F4492Ad5f5Ab9F6 -a 0x2fBe606e211b1BFD0ffE53aa7e15d299824a9478 -n echo-dapp -t applications/echo-dapp/

# Oshiro
docker exec c_espresso cartesi-rollups-cli app register -v -c 0x48F176733DBdEc10EC4d1692e98403E0927E869C -a 0x5a205Fcb6947e200615B75C409ac0aa486D77649 -n echo-dapp -t applications/echo-dapp/
