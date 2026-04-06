package main

type Asset struct {
	AssetID     string `json:"assetId"`
	OwnerID     string `json:"ownerId"`
	DocHash     string `json:"docHash"`
	Metadata    string `json:"metadata"`
	CreatedAt   int64  `json:"createdAt"`
	UpdatedAt   int64  `json:"updatedAt"`
	CreatedTxID string `json:"createdTxId"`
	LastTxID    string `json:"lastTxId"`
}

type CreateAssetRequest struct {
	AssetID  string `json:"assetId"`
	OwnerID  string `json:"ownerId"`
	DocHash  string `json:"docHash"`
	Metadata string `json:"metadata"`
}

type TransferOwnershipRequest struct {
	AssetID    string `json:"assetId"`
	NewOwnerID string `json:"newOwnerId"`
}

type VerifyDocHashRequest struct {
	ProvidedHash string `json:"providedHash"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
