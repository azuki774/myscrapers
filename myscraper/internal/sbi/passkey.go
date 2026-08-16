package sbi

import (
	"encoding/json"
	"fmt"
	"os"
)

// PasskeyCredential is a single WebAuthn credential exported from a
// virtual authenticator (WebAuthn.getCredentials) and stored in the
// passkey file. String fields are base64-encoded, exactly as the CDP
// layer produced them.
type PasskeyCredential struct {
	CredentialID         string `json:"credentialId"`
	IsResidentCredential bool   `json:"isResidentCredential"`
	RPID                 string `json:"rpId"`
	PrivateKey           string `json:"privateKey"`
	UserHandle           string `json:"userHandle"`
	SignCount            int    `json:"signCount"`
}

// PasskeyFile is the on-disk shape of the saved passkey bundle.
type PasskeyFile struct {
	Credentials []PasskeyCredential `json:"credentials"`
}

// LoadPasskey reads and validates a saved passkey JSON file.
func LoadPasskey(path string) (*PasskeyFile, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read passkey file: %w", err)
	}
	var pf PasskeyFile
	if err := json.Unmarshal(raw, &pf); err != nil {
		return nil, fmt.Errorf("parse passkey file: %w", err)
	}
	if len(pf.Credentials) == 0 {
		return nil, fmt.Errorf("passkey file contains no credentials")
	}
	for _, c := range pf.Credentials {
		if c.CredentialID == "" || c.PrivateKey == "" {
			return nil, fmt.Errorf("passkey file contains an incomplete credential")
		}
	}
	return &pf, nil
}
