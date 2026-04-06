package main

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"

	"github.com/hyperledger/fabric-gateway/pkg/client"
	"github.com/hyperledger/fabric-gateway/pkg/identity"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type GatewayConfig struct {
	MSPID          string
	CertPath       string
	KeyPath        string
	PeerEndpoint   string
	TLSCertPath    string
	PeerServerName string
	ChannelName    string
	ChaincodeName  string
}

type GatewayClient struct {
	contract *client.Contract
	gateway  *client.Gateway
	conn     *grpc.ClientConn
}

func NewGatewayClient(cfg GatewayConfig) (*GatewayClient, error) {
	if cfg.MSPID == "" || cfg.CertPath == "" || cfg.KeyPath == "" || cfg.PeerEndpoint == "" || cfg.TLSCertPath == "" || cfg.ChannelName == "" || cfg.ChaincodeName == "" {
		return nil, errors.New("all gateway configuration values must be provided")
	}

	id, err := newIdentity(cfg.MSPID, cfg.CertPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create identity: %w", err)
	}

	sign, err := newSign(cfg.KeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create signing identity: %w", err)
	}

	creds, err := newTLSCredentials(cfg.TLSCertPath, cfg.PeerServerName)
	if err != nil {
		return nil, fmt.Errorf("failed to create TLS credentials: %w", err)
	}

	conn, err := grpc.Dial(cfg.PeerEndpoint, grpc.WithTransportCredentials(creds))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to peer endpoint: %w", err)
	}

	gateway, err := client.Connect(id, client.WithSign(sign), client.WithClientConnection(conn))
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to connect gateway: %w", err)
	}

	network := gateway.GetNetwork(cfg.ChannelName)
	contract := network.GetContract(cfg.ChaincodeName)

	return &GatewayClient{contract: contract, gateway: gateway, conn: conn}, nil
}

func (g *GatewayClient) Close() error {
	if g.gateway != nil {
		if err := g.gateway.Close(); err != nil {
			return err
		}
	}
	if g.conn != nil {
		return g.conn.Close()
	}
	return nil
}

func (g *GatewayClient) SubmitTransaction(functionName string, args ...string) ([]byte, error) {
	return g.contract.SubmitTransaction(functionName, args...)
}

func (g *GatewayClient) EvaluateTransaction(functionName string, args ...string) ([]byte, error) {
	return g.contract.EvaluateTransaction(functionName, args...)
}

func newIdentity(mspID, certPath string) (*identity.X509Identity, error) {
	certificatePEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read certificate: %w", err)
	}

	block, _ := pem.Decode(certificatePEM)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM certificate")
	}

	certificate, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse x509 certificate: %w", err)
	}

	return identity.NewX509Identity(mspID, certificate)
}

func newSign(keyPath string) (identity.Sign, error) {
	privateKeyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key: %w", err)
	}

	privateKey, err := identity.PrivateKeyFromPEM(privateKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	return identity.NewPrivateKeySign(privateKey)
}

func newTLSCredentials(tlsCertPath string, serverName string) (credentials.TransportCredentials, error) {
	certPEM, err := os.ReadFile(tlsCertPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read TLS certificate: %w", err)
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(certPEM) {
		return nil, errors.New("failed to append TLS certificate to pool")
	}

	return credentials.NewClientTLSFromCert(certPool, serverName), nil
}
