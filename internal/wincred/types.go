package wincred

import (
	"golang.org/x/sys/windows"
)

const (
	// CredTypeGeneric defines a generic credential in Windows Credential Manager.
	CredTypeGeneric = 1

	// CredPersistLocalMachine persists credentials across reboots for the local machine.
	CredPersistLocalMachine = 2

	// DefaultTargetName is Antigravity's target in Windows Credential Manager.
	DefaultTargetName = "LegacyGeneric:target=gemini:antigravity"

	// DefaultUserName is the default username associated with Antigravity credentials.
	DefaultUserName = "antigravity"
)

// CREDENTIALW maps directly to the Windows API structure CREDENTIALW in Advapi32.dll.
type CREDENTIALW struct {
	Flags              uint32
	Type               uint32
	TargetName         *uint16
	Comment            *uint16
	LastWritten        windows.Filetime
	CredentialBlobSize uint32
	CredentialBlob     *byte
	Persist            uint32
	AttributeCount     uint32
	Attributes         uintptr
	TargetAlias        *uint16
	UserName           *uint16
}
