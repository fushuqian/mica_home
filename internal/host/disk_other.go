//go:build !linux

package host

// diskUsage 在非 Linux 平台不返回容量信息。
func diskUsage(path string) (free, total int64) {
	return 0, 0
}
