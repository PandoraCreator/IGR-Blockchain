package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
)

func main() {
	config := GatewayConfig{
		MSPID:          envOrDefault("FABRIC_MSPID", "Org1MSP"),
		CertPath:       envOrDefault("FABRIC_CERT_PATH", ""),
		KeyPath:        envOrDefault("FABRIC_KEY_PATH", ""),
		TLSCertPath:    envOrDefault("FABRIC_TLS_CERT_PATH", ""),
		PeerEndpoint:   envOrDefault("FABRIC_PEER_ENDPOINT", "localhost:7051"),
		PeerServerName: envOrDefault("FABRIC_PEER_SERVER_NAME", "peer0.org1.example.com"),
		ChannelName:    envOrDefault("FABRIC_CHANNEL_NAME", "mychannel"),
		ChaincodeName:  envOrDefault("FABRIC_CHAINCODE_NAME", "asset-registry"),
	}

	gatewayClient, err := NewGatewayClient(config)
	if err != nil {
		log.Fatalf("failed to initialize gateway: %v", err)
	}
	defer gatewayClient.Close()

	router := mux.NewRouter()
	RegisterHandlers(router, gatewayClient)

	listenAddr := envOrDefault("API_LISTEN_ADDR", ":8080")
	log.Printf("API server listening on %s", listenAddr)
	if err := http.ListenAndServe(listenAddr, router); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func envOrDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
