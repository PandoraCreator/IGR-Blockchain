#!/bin/bash
set -e

echo "=========================================="
echo "Asset Registry Chaincode Deployment"
echo "=========================================="

SCRIPT_DIR="$( cd "$( dirname "$0" )" && pwd )"
TEST_NETWORK_DIR="$SCRIPT_DIR/fabric-samples/test-network"
CHAINCODE_DIR="$SCRIPT_DIR/asset-registry"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# Step 1: Check prerequisites
info "Checking prerequisites..."

if [ ! -d "$TEST_NETWORK_DIR" ]; then
    error "test-network directory not found at $TEST_NETWORK_DIR"
fi

if [ ! -f "$TEST_NETWORK_DIR/network.sh" ]; then
    error "network.sh not found"
fi

if [ ! -d "$CHAINCODE_DIR" ]; then
    error "asset-registry chaincode directory not found at $CHAINCODE_DIR"
fi

# Step 2: Check if network is running
info "Checking if Fabric network is running..."
if ! nc -z localhost 7050 2>/dev/null; then
    warning "Orderer (port 7050) is not responding. Attempting to start network..."
    cd "$TEST_NETWORK_DIR"
    info "Starting Fabric network with channel..."
    ./network.sh up createChannel || error "Failed to start network"
    cd "$SCRIPT_DIR"
fi

info "Network is running (port 7050 responsive)"

# Step 3: Deploy chaincode
cd "$TEST_NETWORK_DIR"
info "Deploying asset-registry chaincode..."

./network.sh deployCC \
    -ccn asset-registry \
    -ccp ../../asset-registry \
    -ccl go \
    -ccv 1.0 || error "Deployment failed"

info "Chaincode deployed successfully"

# Step 4: Verify deployment
export FABRIC_CFG_PATH=$PWD/../config
source ./scripts/envVar.sh 2>/dev/null || true
export PATH=$PWD/../bin:$PATH

info "Verifying chaincode deployment..."
if peer lifecycle chaincode querycommitted --channelID mychannel --output json 2>/dev/null | grep -q "asset-registry"; then
    info "Chaincode 'asset-registry' verified on mychannel"
else
    error "Chaincode verification failed"
fi

# Step 5: Test the chaincode
echo ""
echo "=========================================="
echo "Testing Chaincode"
echo "=========================================="

peer chaincode query -C mychannel -n asset-registry \
    -c '{"function":"ListAllAssets","Args":[]}' 2>&1 || warning "Initial query may show empty results"

echo ""
echo "=========================================="
echo "Deployment Complete!"
echo "=========================================="
info "Asset Registry chaincode is ready"
echo ""
echo "Quick Test Commands:"
echo "===================="
echo ""
echo "1. Create Asset (Org1 only):"
echo "   peer chaincode invoke -C mychannel -n asset-registry \\"
echo "     -c '{\"function\":\"CreateAsset\",\"Args\":[\"asset-001\",\"owner1\",\"hash123\",\"metadata\"]}'"
echo ""
echo "2. Read Asset (Both Orgs):"
echo "   peer chaincode query -C mychannel -n asset-registry \\"
echo "     -c '{\"function\":\"ReadAsset\",\"Args\":[\"asset-001\"]}'"
echo ""
echo "See DEPLOYMENT_INSTRUCTIONS.md for more details"
echo ""
