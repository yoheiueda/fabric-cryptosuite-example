
dir=$(dirname "$0")
cd "$dir/.."
rm -fr ./crypto-config
./bin/cryptogen generate --config ./crypto-config.yaml --output ./crypto-config
key=$(basename $(ls crypto-config/peerOrganizations/org1.example.com/ca/*_sk | tail -1))
sed -i.old "s,FABRIC_CA_SERVER_CA_KEYFILE=/etc/hyperledger/fabric-ca-server-config/.*_sk,FABRIC_CA_SERVER_CA_KEYFILE=/etc/hyperledger/fabric-ca-server-config/$key," ../docker-compose.yaml
./bin/configtxgen -profile OrdererGenesis -channelID testchainid -outputBlock ./genesis.block
./bin/configtxgen -profile MyChannel -channelID mychannel -outputCreateChannelTx ./mychannel.tx
