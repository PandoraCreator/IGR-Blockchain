package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"github.com/gorilla/mux"
)

func RegisterHandlers(router *mux.Router, gateway *GatewayClient) {
	router.HandleFunc("/asset", createAssetHandler(gateway)).Methods(http.MethodPost)
	router.HandleFunc("/asset/transfer", transferOwnershipHandler(gateway)).Methods(http.MethodPost)
	router.HandleFunc("/asset/{assetId}", readAssetHandler(gateway)).Methods(http.MethodGet)
	router.HandleFunc("/assets", listAllAssetsHandler(gateway)).Methods(http.MethodGet)
	router.HandleFunc("/assets/owner/{ownerId}", getAssetsByOwnerHandler(gateway)).Methods(http.MethodGet)
	router.HandleFunc("/asset/{assetId}/verify", verifyDocHashHandler(gateway)).Methods(http.MethodPost)
}

func createAssetHandler(gateway *GatewayClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req CreateAssetRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("invalid request body: %w", err))
			return
		}

		if req.AssetID == "" || req.OwnerID == "" || req.DocHash == "" {
			writeError(w, http.StatusBadRequest, fmt.Errorf("assetId, ownerId and docHash are required"))
			return
		}

		_, err := gateway.SubmitTransaction("CreateAsset", req.AssetID, req.OwnerID, req.DocHash, req.Metadata)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		writeJSON(w, http.StatusCreated, map[string]string{"message": "asset created"})
	}
}

func transferOwnershipHandler(gateway *GatewayClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req TransferOwnershipRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("invalid request body: %w", err))
			return
		}

		if req.AssetID == "" || req.NewOwnerID == "" {
			writeError(w, http.StatusBadRequest, fmt.Errorf("assetId and newOwnerId are required"))
			return
		}

		_, err := gateway.SubmitTransaction("TransferOwnership", req.AssetID, req.NewOwnerID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"message": "ownership transferred"})
	}
}

func readAssetHandler(gateway *GatewayClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		assetID := vars["assetId"]
		if assetID == "" {
			writeError(w, http.StatusBadRequest, fmt.Errorf("assetId is required"))
			return
		}

		result, err := gateway.EvaluateTransaction("ReadAsset", assetID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		var asset Asset
		if err := json.Unmarshal(result, &asset); err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Errorf("failed to parse asset response: %w", err))
			return
		}

		writeJSON(w, http.StatusOK, asset)
	}
}

func listAllAssetsHandler(gateway *GatewayClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result, err := gateway.EvaluateTransaction("ListAllAssets")
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		var assets []Asset
		if err := json.Unmarshal(result, &assets); err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Errorf("failed to parse assets response: %w", err))
			return
		}

		writeJSON(w, http.StatusOK, assets)
	}
}

func getAssetsByOwnerHandler(gateway *GatewayClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		ownerID := vars["ownerId"]
		if ownerID == "" {
			writeError(w, http.StatusBadRequest, fmt.Errorf("ownerId is required"))
			return
		}

		result, err := gateway.EvaluateTransaction("GetAssetsByOwner", ownerID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		var assets []Asset
		if err := json.Unmarshal(result, &assets); err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Errorf("failed to parse assets response: %w", err))
			return
		}

		writeJSON(w, http.StatusOK, assets)
	}
}

func verifyDocHashHandler(gateway *GatewayClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		assetID := vars["assetId"]
		if assetID == "" {
			writeError(w, http.StatusBadRequest, fmt.Errorf("assetId is required"))
			return
		}

		var req VerifyDocHashRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("invalid request body: %w", err))
			return
		}

		if req.ProvidedHash == "" {
			writeError(w, http.StatusBadRequest, fmt.Errorf("providedHash is required"))
			return
		}

		result, err := gateway.EvaluateTransaction("VerifyDocHash", assetID, req.ProvidedHash)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		var verified bool
		if err := json.Unmarshal(result, &verified); err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Errorf("failed to parse verify response: %w", err))
			return
		}

		writeJSON(w, http.StatusOK, map[string]bool{"verified": verified})
	}
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, ErrorResponse{Error: err.Error()})
}
