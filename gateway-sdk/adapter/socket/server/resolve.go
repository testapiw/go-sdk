package server

import (
	"fmt"
	"os/user"
	"strconv"
)

// resolveIDs transforms user and group names into numeric UID and GID.
// Returns -1 for empty strings to instruct os.Chown to preserve current ownership.
func resolveIDs(userName, groupName string) (int, int, error) {
	uid, gid := -1, -1

	if userName != "" {
		u, err := user.Lookup(userName)
		if err != nil {
			return -1, -1, fmt.Errorf("lookup user %q: %w", userName, err)
		}
		parsedUID, err := strconv.Atoi(u.Uid)
		if err != nil {
			return -1, -1, fmt.Errorf("parse uid %q: %w", u.Uid, err)
		}
		uid = parsedUID
	}

	if groupName != "" {
		g, err := user.LookupGroup(groupName)
		if err != nil {
			return -1, -1, fmt.Errorf("lookup group %q: %w", groupName, err)
		}
		parsedGID, err := strconv.Atoi(g.Gid)
		if err != nil {
			return -1, -1, fmt.Errorf("parse gid %q: %w", g.Gid, err)
		}
		gid = parsedGID
	}

	return uid, gid, nil
}
