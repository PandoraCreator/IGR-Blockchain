# Asset Registry REST API

This service provides HTTP endpoints for the `asset-registry` chaincode.

## Setup

1. Set the Fabric gateway environment variables:

- `FABRIC_MSPID` (default: `Org1MSP`)
- `FABRIC_CERT_PATH` - path to the client certificate PEM file
- `FABRIC_KEY_PATH` - path to the client private key PEM file
- `FABRIC_TLS_CERT_PATH` - path to the peer TLS certificate PEM file
- `FABRIC_PEER_ENDPOINT` - peer endpoint, e.g. `localhost:7051`
- `FABRIC_CHANNEL_NAME` - Fabric channel name, e.g. `mychannel`
- `FABRIC_CHAINCODE_NAME` - chaincode name, e.g. `asset-registry`
- `API_LISTEN_ADDR` - HTTP listen address (default `:8080`)

2. Run the service:

```bash
cd asset-registry/api
go run ./...
```

## Endpoints

### Create asset
POST `/asset`

Body:
```json
{
  "assetId": "asset-001",
  "ownerId": "owner1",
  "docHash": "hash123",
  "metadata": "some metadata"
}
```

### Transfer ownership
POST `/asset/transfer`

Body:
```json
{
  "assetId": "asset-001",
  "newOwnerId": "owner2"
}
```

### Read asset
GET `/asset/{assetId}`

### List all assets
GET `/assets`

### Get assets by owner
GET `/assets/owner/{ownerId}`

### Verify document hash
POST `/asset/{assetId}/verify`

Body:
```json
{
  "providedHash": "hash123"
}
```
