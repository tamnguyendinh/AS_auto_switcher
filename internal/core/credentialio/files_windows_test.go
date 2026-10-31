//go:build windows

package credentialio

import (
	"bytes"
	"golang.org/x/sys/windows"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWritePreservesOldFileWhenWindowsReplaceDenied(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "auth.json")
	old := []byte("old credential")
	if err := Write(path, old); err != nil {
		t.Fatal(err)
	}
	name, _ := windows.UTF16PtrFromString(path)
	// Open without FILE_SHARE_DELETE to force the real Windows replace to fail.
	handle, err := windows.CreateFile(name, windows.GENERIC_READ, windows.FILE_SHARE_READ, nil, windows.OPEN_EXISTING, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	err = Write(path, []byte("new credential"))
	windows.CloseHandle(handle)
	if err == nil {
		t.Fatal("replace unexpectedly succeeded under a delete-sharing lock")
	}
	got, err := Read(path)
	if err != nil || !bytes.Equal(got, old) {
		t.Fatal("failed replace destroyed old credential")
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatal("failed replace leaked temporary credential")
	}
}

func TestPrivateCredentialACLAndRoundTrip(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "private")
	if err := PrivateDir(dir); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "auth.json")
	for _, data := range [][]byte{[]byte("first"), []byte("second")} {
		if err := Write(path, data); err != nil {
			t.Fatal(err)
		}
		got, err := Read(path)
		if err != nil || !bytes.Equal(got, data) {
			t.Fatal("credential did not round-trip")
		}
	}
	for _, target := range []string{dir, path} {
		sd, err := windows.GetNamedSecurityInfo(target, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
		if err != nil {
			t.Fatal(err)
		}
		control, _, err := sd.Control()
		if err != nil || control&windows.SE_DACL_PROTECTED == 0 {
			t.Fatal("credential DACL inherits public access")
		}
		user, err := windows.GetCurrentProcessToken().GetTokenUser()
		if err != nil {
			t.Fatal(err)
		}
		sddl := sd.String()
		if !strings.Contains(sddl, user.User.Sid.String()) || strings.Count(sddl, "(A;") != 1 {
			t.Fatal("credential ACL does not restrict access to current user")
		}
	}
}

func TestReparseCredentialDestinationRejected(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	link := filepath.Join(root, "linked")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("Windows symlink privilege unavailable: %v", err)
	}
	if err := Write(filepath.Join(link, "auth.json"), []byte("secret")); err == nil {
		t.Fatal("write followed reparse destination")
	}
	if _, err := os.Stat(filepath.Join(outside, "auth.json")); !os.IsNotExist(err) {
		t.Fatal("outside directory changed")
	}
}
