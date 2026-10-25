//go:build linux

package backup

import "golang.org/x/sys/unix"

// The destination must not exist, including during the final atomic operation.
func publishDirectory(stage, dest string) error {
	return unix.Renameat2(unix.AT_FDCWD, stage, unix.AT_FDCWD, dest, unix.RENAME_NOREPLACE)
}
