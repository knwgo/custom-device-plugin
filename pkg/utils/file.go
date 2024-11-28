package utils

import (
	"fmt"
	"os"
	"syscall"
)

func GetFileInode(fi os.FileInfo) (uint64, error) {
	stat, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, fmt.Errorf("cannot get file inode")
	}

	return stat.Ino, nil
}
