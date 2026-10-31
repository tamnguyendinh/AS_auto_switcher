package wincred

import (
	"encoding/json"

	corewincred "as-auto-switcher/internal/core/wincred"
	"as-auto-switcher/internal/engines/antigravity/model"
)

const (
	// DefaultTargetName is Antigravity's target in Windows Credential Manager.
	DefaultTargetName = "LegacyGeneric:target=gemini:antigravity"
	// DefaultUserName is the default username associated with Antigravity credentials.
	DefaultUserName = "antigravity"
)

var (
	// Keep the raw layer's sentinel identities for existing errors.Is callers.
	ErrNotFound       = corewincred.ErrNotFound
	ErrInvalidPayload = corewincred.ErrInvalidPayload
)

// ReadToken reads and parses the active Antigravity OAuth token from the default target.
func ReadToken() (*model.TokenData, error) {
	return ReadTokenTarget(DefaultTargetName)
}

// ReadTokenTarget reads and parses token data from a specified target.
func ReadTokenTarget(target string) (*model.TokenData, error) {
	raw, err := corewincred.ReadRaw(target)
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return nil, ErrNotFound
	}

	var token model.TokenData
	if err := json.Unmarshal(raw, &token); err != nil {
		return nil, err
	}
	if !token.IsValid() {
		return nil, ErrInvalidPayload
	}

	return &token, nil
}

// WriteToken serializes and writes token data to Antigravity's default credential target.
func WriteToken(token *model.TokenData) error {
	return WriteTokenTarget(DefaultTargetName, DefaultUserName, token)
}

// WriteTokenTarget serializes and writes token data to a specific target and username.
func WriteTokenTarget(target string, username string, token *model.TokenData) error {
	if token == nil || !token.IsValid() {
		return ErrInvalidPayload
	}

	blob, err := token.MarshalJSON()
	if err != nil {
		return err
	}
	if len(blob) == 0 {
		return ErrInvalidPayload
	}

	return corewincred.WriteRaw(target, username, blob)
}

// DeleteToken safely deletes the Antigravity credential from Windows Credential Manager.
func DeleteToken() error {
	return DeleteTokenTarget(DefaultTargetName)
}

// DeleteTokenTarget deletes the credential for a specific target.
func DeleteTokenTarget(target string) error {
	return corewincred.DeleteRaw(target)
}

// HasToken checks whether a valid Antigravity token exists in Windows Credential Manager.
func HasToken() bool {
	return HasTokenTarget(DefaultTargetName)
}

// HasTokenTarget checks whether a valid token exists for a specific target.
func HasTokenTarget(target string) bool {
	token, err := ReadTokenTarget(target)
	return err == nil && token != nil && token.IsValid()
}
