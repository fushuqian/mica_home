//go:build linux

package host

import "syscall"

// diskUsage 返回路径所在文件系统的可用空间与总容量（字节）。
func diskUsage(path string) (free, total int64) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return 0, 0
	}
	return int64(st.Bavail) * int64(st.Bsize), int64(st.Blocks) * int64(st.Bsize)
}
