package identity

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	libp2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
)

// LoadOrCreate loads the identity key from keyPath, creating it if missing.
// Returns the private key and the derived libp2p peer ID string.
func LoadOrCreate(keyPath string) (libp2pcrypto.PrivKey, string, error) {
	if err := os.MkdirAll(filepath.Dir(keyPath), 0700); err != nil {
		return nil, "", fmt.Errorf("creating identity dir: %w", err)
	}

	// Try loading existing key.
	if data, err := os.ReadFile(keyPath); err == nil {
		hexStr := strings.TrimSpace(string(data))
		keyBytes, decErr := hex.DecodeString(hexStr)
		if decErr == nil {
			privKey, unmarshalErr := libp2pcrypto.UnmarshalPrivateKey(keyBytes)
			if unmarshalErr == nil {
				peerID, pidErr := peer.IDFromPrivateKey(privKey)
				if pidErr == nil {
					return privKey, peerID.String(), nil
				}
			}
		}
	}

	// Generate a new secp256k1 keypair.
	privKey, _, err := libp2pcrypto.GenerateKeyPair(libp2pcrypto.Secp256k1, 256)
	if err != nil {
		return nil, "", fmt.Errorf("generating key pair: %w", err)
	}

	keyBytes, err := libp2pcrypto.MarshalPrivateKey(privKey)
	if err != nil {
		return nil, "", fmt.Errorf("marshaling key: %w", err)
	}

	if err := os.WriteFile(keyPath, []byte(hex.EncodeToString(keyBytes)), 0600); err != nil {
		return nil, "", fmt.Errorf("saving identity key: %w", err)
	}

	peerID, err := peer.IDFromPrivateKey(privKey)
	if err != nil {
		return nil, "", fmt.Errorf("deriving peer ID: %w", err)
	}

	return privKey, peerID.String(), nil
}
