//go:build !linux

package backup

import "errors"

func publishDirectory(stage, dest string) error {
	return errors.New("此平台尚未提供安全原子恢复，请使用Debian/Linux恢复到新目录")
}
