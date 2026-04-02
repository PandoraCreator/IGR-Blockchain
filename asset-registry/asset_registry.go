package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

// Asset represents a registered asset with ownership and document information
type Asset struct {
	AssetID      string `json:"assetId"`
	OwnerID      string `json:"ownerId"`
	DocHash      string `json:"docHash"`
	Metadata     string `json:"metadata"`
	CreatedAt    int64  `json:"createdAt"`
	UpdatedAt    int64  `json:"updatedAt"`
	CreatedTxID  string `json:"createdTxId"`
	LastTxID     string `json:"lastTxId"`
}

// AssetRegistryChaincode handles the asset registry smart contract
type AssetRegistryChaincode struct {
	contractapi.Contract
}

// OwnershipTransferred event emitted when ownership changes
type OwnershipTransferred struct {
	AssetID   string `json:"assetId"`
	FromOwner string `json:"fromOwner"`
	ToOwner   string `json:"toOwner"`
	TxID      string `json:"txId"`
	Timestamp int64  `json:"timestamp"`
}

// CreateAsset creates a new asset (Org1 only)
// Args: assetId, ownerId, docHash, metadata
func (cc *AssetRegistryChaincode) CreateAsset(ctx contractapi.TransactionContextInterface, assetID string, ownerID string, docHash string, metadata string) error {
	// RBAC: Only Org1MSP can create assets
	mspID, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return fmt.Errorf("failed to get MSP ID: %v", err)
	}

	if mspID != "Org1MSP" {
		return fmt.Errorf("only Org1MSP can create assets, got %s", mspID)
	}

	// Check if asset already exists
	existingAsset, err := ctx.GetStub().GetState(assetID)
	if err != nil {
		return fmt.Errorf("failed to read from world state: %v", err)
	}

	if existingAsset != nil {
		return fmt.Errorf("asset with ID %s already exists", assetID)
	}

	// Validate required fields
	if assetID == "" || ownerID == "" || docHash == "" {
		return fmt.Errorf("assetId, ownerId, and docHash cannot be empty")
	}

	// Create new asset
	now := time.Now().UnixMilli()
	txID := ctx.GetStub().GetTxID()

	asset := Asset{
		AssetID:     assetID,
		OwnerID:     ownerID,
		DocHash:     docHash,
		Metadata:    metadata,
		CreatedAt:   now,
		UpdatedAt:   now,
		CreatedTxID: txID,
		LastTxID:    txID,
	}

	// Marshal to JSON
	assetJSON, err := json.Marshal(asset)
	if err != nil {
		return fmt.Errorf("failed to marshal asset: %v", err)
	}

	// Store in world state
	err = ctx.GetStub().PutState(assetID, assetJSON)
	if err != nil {
		return fmt.Errorf("failed to put state: %v", err)
	}

	// Emit event
	event := map[string]interface{}{
		"eventType": "AssetCreated",
		"assetId":   assetID,
		"ownerId":   ownerID,
		"docHash":   docHash,
		"txId":      txID,
		"timestamp": now,
	}

	eventJSON, _ := json.Marshal(event)
	ctx.GetStub().SetEvent("AssetCreated", eventJSON)

	return nil
}

// ReadAsset reads an asset (both orgs can read)
// Args: assetId
func (cc *AssetRegistryChaincode) ReadAsset(ctx contractapi.TransactionContextInterface, assetID string) (*Asset, error) {
	// Both Org1MSP and Org2MSP can read
	assetJSON, err := ctx.GetStub().GetState(assetID)
	if err != nil {
		return nil, fmt.Errorf("failed to read from world state: %v", err)
	}

	if assetJSON == nil {
		return nil, fmt.Errorf("asset with ID %s does not exist", assetID)
	}

	var asset Asset
	err = json.Unmarshal(assetJSON, &asset)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal asset: %v", err)
	}

	return &asset, nil
}

// TransferOwnership transfers ownership of an asset (Org1 only)
// Args: assetId, newOwnerId
func (cc *AssetRegistryChaincode) TransferOwnership(ctx contractapi.TransactionContextInterface, assetID string, newOwnerID string) error {
	// RBAC: Only Org1MSP can transfer ownership
	mspID, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return fmt.Errorf("failed to get MSP ID: %v", err)
	}

	if mspID != "Org1MSP" {
		return fmt.Errorf("only Org1MSP can transfer ownership, got %s", mspID)
	}

	// Validate inputs
	if assetID == "" || newOwnerID == "" {
		return fmt.Errorf("assetId and newOwnerId cannot be empty")
	}

	// Read existing asset
	assetJSON, err := ctx.GetStub().GetState(assetID)
	if err != nil {
		return fmt.Errorf("failed to read from world state: %v", err)
	}

	if assetJSON == nil {
		return fmt.Errorf("asset with ID %s does not exist", assetID)
	}

	var asset Asset
	err = json.Unmarshal(assetJSON, &asset)
	if err != nil {
		return fmt.Errorf("failed to unmarshal asset: %v", err)
	}

	// Store old owner for event
	oldOwnerID := asset.OwnerID

	// Update ownership
	now := time.Now().UnixMilli()
	txID := ctx.GetStub().GetTxID()

	asset.OwnerID = newOwnerID
	asset.UpdatedAt = now
	asset.LastTxID = txID

	// Marshal updated asset
	updatedAssetJSON, err := json.Marshal(asset)
	if err != nil {
		return fmt.Errorf("failed to marshal asset: %v", err)
	}

	// Store updated asset
	err = ctx.GetStub().PutState(assetID, updatedAssetJSON)
	if err != nil {
		return fmt.Errorf("failed to put state: %v", err)
	}

	// Emit OwnershipTransferred event
	ownershipEvent := OwnershipTransferred{
		AssetID:   assetID,
		FromOwner: oldOwnerID,
		ToOwner:   newOwnerID,
		TxID:      txID,
		Timestamp: now,
	}

	eventJSON, _ := json.Marshal(ownershipEvent)
	ctx.GetStub().SetEvent("OwnershipTransferred", eventJSON)

	return nil
}

// ListAllAssets returns all assets (both orgs can read)
func (cc *AssetRegistryChaincode) ListAllAssets(ctx contractapi.TransactionContextInterface) ([]*Asset, error) {
	// Query all assets using key range
	resultsIterator, err := ctx.GetStub().GetStateByRange("", "")
	if err != nil {
		return nil, fmt.Errorf("failed to get state by range: %v", err)
	}
	defer resultsIterator.Close()

	var assets []*Asset
	for resultsIterator.HasNext() {
		result, err := resultsIterator.Next()
		if err != nil {
			return nil, fmt.Errorf("failed to iterate results: %v", err)
		}

		var asset Asset
		err = json.Unmarshal(result.Value, &asset)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal asset: %v", err)
		}

		assets = append(assets, &asset)
	}

	return assets, nil
}

// GetAssetsByOwner returns all assets owned by a specific owner (both orgs can read)
// Args: ownerId
func (cc *AssetRegistryChaincode) GetAssetsByOwner(ctx contractapi.TransactionContextInterface, ownerID string) ([]*Asset, error) {
	// Iterate through all assets and filter by owner
	resultsIterator, err := ctx.GetStub().GetStateByRange("", "")
	if err != nil {
		return nil, fmt.Errorf("failed to get state by range: %v", err)
	}
	defer resultsIterator.Close()

	var assets []*Asset
	for resultsIterator.HasNext() {
		result, err := resultsIterator.Next()
		if err != nil {
			return nil, fmt.Errorf("failed to iterate results: %v", err)
		}

		var asset Asset
		err = json.Unmarshal(result.Value, &asset)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal asset: %v", err)
		}

		if asset.OwnerID == ownerID {
			assets = append(assets, &asset)
		}
	}

	return assets, nil
}

// VerifyDocHash verifies if a provided hash matches the stored hash
// Args: assetId, providedHash
func (cc *AssetRegistryChaincode) VerifyDocHash(ctx contractapi.TransactionContextInterface, assetID string, providedHash string) (bool, error) {
	asset, err := cc.ReadAsset(ctx, assetID)
	if err != nil {
		return false, fmt.Errorf("failed to read asset: %v", err)
	}

	return asset.DocHash == providedHash, nil
}

func main() {
	chaincode, err := contractapi.NewChaincode(&AssetRegistryChaincode{})
	if err != nil {
		log.Panicf("Error creating asset registry chaincode: %v", err)
	}

	if err := chaincode.Start(); err != nil {
		log.Panicf("Error starting asset registry chaincode: %v", err)
	}
}
