// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package internal_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"cloudeng.io/cmd/idu/internal"
	"cloudeng.io/file"
)

func TestUserInfo(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFileName := filepath.Join(tmpDir, "a")
	if _, err := os.Create(tmpFileName); err != nil {
		t.Fatal(err)
	}
	info, err := os.DirFS(tmpDir).(fs.StatFS).Stat("a")
	if err != nil {
		t.Fatal(err)
	}

	fi := file.NewInfoFromFileInfo(info)

	pi := internal.PrefixInfo{
		UserID:  1,
		GroupID: 2,
	}

	uid, gid := pi.GetUserGroup(fi)

	if got, want := int(uid), os.Getuid(); got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := int(gid), os.Getgid(); got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	pi.SetUserGroup(&fi, 600, 6)

	uid, gid = pi.GetUserGroup(fi)
	if got, want := uid, uint32(600); got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	if got, want := gid, uint32(6); got != want {
		t.Errorf("got %v, want %v", got, want)
	}

}