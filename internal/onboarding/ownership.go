package onboarding

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func hostOwnershipFromEnv() (uid, gid int, ok bool, err error) {
	return parseHostOwnership(os.Getenv("HOST_UID"), os.Getenv("HOST_GID"))
}

func parseHostOwnership(uidRaw, gidRaw string) (uid, gid int, ok bool, err error) {
	uidRaw = strings.TrimSpace(uidRaw)
	gidRaw = strings.TrimSpace(gidRaw)
	if uidRaw == "" && gidRaw == "" {
		return 0, 0, false, nil
	}
	if uidRaw == "" || gidRaw == "" {
		return 0, 0, false, fmt.Errorf("HOST_UID and HOST_GID must both be set")
	}
	uid, err = strconv.Atoi(uidRaw)
	if err != nil {
		return 0, 0, false, fmt.Errorf("parse HOST_UID %q: %w", uidRaw, err)
	}
	gid, err = strconv.Atoi(gidRaw)
	if err != nil {
		return 0, 0, false, fmt.Errorf("parse HOST_GID %q: %w", gidRaw, err)
	}
	if uid < 0 || gid < 0 {
		return 0, 0, false, fmt.Errorf("HOST_UID and HOST_GID must be non-negative")
	}
	return uid, gid, true, nil
}

func applyHostOwnership(repoRoot, path string) error {
	if os.Geteuid() != 0 {
		return nil
	}
	uid, gid, ok, err := hostOwnershipFromEnv()
	if err != nil || !ok {
		return err
	}
	root := filepath.Clean(repoRoot)
	current := filepath.Clean(path)
	for {
		if err := os.Chown(current, uid, gid); err != nil {
			return fmt.Errorf("chown %s: %w", current, err)
		}
		if current == root {
			return nil
		}
		parent := filepath.Dir(current)
		if parent == current || !strings.HasPrefix(parent, root) {
			return nil
		}
		current = parent
	}
}
