// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package internal

import "cloudeng.io/file"

func UserInfo(fi file.Info) (userID, groupID uint32, ok bool) {
	u, g, ok := userGroupID(fi)
	return u, g, ok
}