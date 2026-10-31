//go:build windows

// Package credentialio owns private, durable credential file operations.
package credentialio

import (
	"errors"
	"golang.org/x/sys/windows"
	"io"
	"os"
	"path/filepath"
)

// CheckPath rejects reparse points in every existing path component, including
// junctions on parents. Missing final components are permitted for creation.
func CheckPath(path string) error {
	path, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	for {
		p, err := windows.UTF16PtrFromString(path)
		if err != nil {
			return err
		}
		attrs, err := windows.GetFileAttributes(p)
		if err == nil && attrs&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
			return errors.New("credential path contains a reparse point")
		}
		if err != nil && !errors.Is(err, windows.ERROR_FILE_NOT_FOUND) && !errors.Is(err, windows.ERROR_PATH_NOT_FOUND) {
			return err
		}
		parent := filepath.Dir(path)
		if parent == path {
			return nil
		}
		path = parent
	}
}

func Protect(path string, directory bool) error {
	if err := CheckPath(path); err != nil {
		return err
	}
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return err
	}
	flags := ""
	if directory {
		flags = "OICI"
	}
	sd, err := windows.SecurityDescriptorFromString("D:P(A;" + flags + ";FA;;;" + user.User.Sid.String() + ")")
	if err != nil {
		return err
	}
	acl, _, err := sd.DACL()
	if err != nil {
		return err
	}
	return windows.SetNamedSecurityInfo(path, windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, acl, nil)
}

func PrivateDir(path string) error {
	if err := CheckPath(path); err != nil {
		return err
	}
	if err := os.MkdirAll(path, 0700); err != nil {
		return err
	}
	return Protect(path, true)
}

func Read(path string) ([]byte, error) {
	if err := CheckPath(path); err != nil {
		return nil, err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, errors.New("credential is not a regular file")
	}
	return os.ReadFile(path)
}

// Write never removes the old destination to work around a failed replace.
// The temporary file is private before any secret is written.
func Write(path string, data []byte) error {
	if err := CheckPath(path); err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, "credential_*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	defer f.Close()
	if err := Protect(f.Name(), false); err != nil {
		return err
	}
	if n, err := f.Write(data); err != nil {
		return err
	} else if n != len(data) {
		return io.ErrShortWrite
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := CheckPath(path); err != nil {
		return err
	}
	from, err := windows.UTF16PtrFromString(f.Name())
	if err != nil {
		return err
	}
	to, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	if err := windows.MoveFileEx(from, to, windows.MOVEFILE_REPLACE_EXISTING|windows.MOVEFILE_WRITE_THROUGH); err != nil {
		return &os.PathError{Op: "replace", Path: path, Err: err}
	}
	return nil
}
