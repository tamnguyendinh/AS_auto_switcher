package wincred

import (
	"errors"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modAdvapi32     = windows.NewLazySystemDLL("advapi32.dll")
	procCredReadW   = modAdvapi32.NewProc("CredReadW")
	procCredWriteW  = modAdvapi32.NewProc("CredWriteW")
	procCredDeleteW = modAdvapi32.NewProc("CredDeleteW")
	procCredFree    = modAdvapi32.NewProc("CredFree")

	// ErrNotFound indicates that the requested credential was not found in Windows Credential Manager.
	ErrNotFound = errors.New("credential not found")

	// ErrInvalidPayload indicates that the credential blob or token structure is invalid or empty.
	ErrInvalidPayload = errors.New("invalid or empty token payload")
)

// ReadRaw reads the raw CredentialBlob bytes for a given target name.
func ReadRaw(target string) ([]byte, error) {
	if strings.TrimSpace(target) == "" {
		return nil, errors.New("target name cannot be empty")
	}

	targetPtr, err := windows.UTF16PtrFromString(target)
	if err != nil {
		return nil, err
	}

	var pCred *CREDENTIALW
	r1, _, lastErr := procCredReadW.Call(
		uintptr(unsafe.Pointer(targetPtr)),
		uintptr(CredTypeGeneric),
		0,
		uintptr(unsafe.Pointer(&pCred)),
	)

	if r1 == 0 {
		if errors.Is(lastErr, windows.ERROR_NOT_FOUND) {
			return nil, ErrNotFound
		}
		return nil, lastErr
	}

	defer procCredFree.Call(uintptr(unsafe.Pointer(pCred)))

	if pCred == nil || pCred.CredentialBlobSize == 0 || pCred.CredentialBlob == nil {
		return []byte{}, nil
	}

	raw := make([]byte, pCred.CredentialBlobSize)
	copy(raw, unsafe.Slice(pCred.CredentialBlob, pCred.CredentialBlobSize))
	return raw, nil
}

// WriteRaw writes raw bytes into Windows Credential Manager for a given target and username.
func WriteRaw(target string, username string, blob []byte) error {
	if strings.TrimSpace(target) == "" {
		return errors.New("target name cannot be empty")
	}
	if len(blob) == 0 {
		return ErrInvalidPayload
	}

	targetPtr, err := windows.UTF16PtrFromString(target)
	if err != nil {
		return err
	}

	userPtr, err := windows.UTF16PtrFromString(username)
	if err != nil {
		return err
	}

	cred := CREDENTIALW{
		Flags:              0,
		Type:               CredTypeGeneric,
		TargetName:         targetPtr,
		Comment:            nil,
		CredentialBlobSize: uint32(len(blob)),
		CredentialBlob:     &blob[0],
		Persist:            CredPersistLocalMachine,
		AttributeCount:     0,
		Attributes:         0,
		TargetAlias:        nil,
		UserName:           userPtr,
	}

	r1, _, lastErr := procCredWriteW.Call(
		uintptr(unsafe.Pointer(&cred)),
		0,
	)

	if r1 == 0 {
		return lastErr
	}

	return nil
}

// DeleteRaw deletes a credential from Windows Credential Manager.
// Deleting a non-existent target returns nil (idempotent operation).
func DeleteRaw(target string) error {
	if strings.TrimSpace(target) == "" {
		return nil
	}

	targetPtr, err := windows.UTF16PtrFromString(target)
	if err != nil {
		return err
	}

	r1, _, lastErr := procCredDeleteW.Call(
		uintptr(unsafe.Pointer(targetPtr)),
		uintptr(CredTypeGeneric),
		0,
	)

	if r1 == 0 {
		if errors.Is(lastErr, windows.ERROR_NOT_FOUND) {
			return nil
		}
		return lastErr
	}

	return nil
}

// HasRaw checks whether a valid, non-empty credential exists for target.
func HasRaw(target string) bool {
	raw, err := ReadRaw(target)
	return err == nil && len(raw) > 0
}
