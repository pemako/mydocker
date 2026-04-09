//go:build !linux

package cgroups

// IsCgroup2UnifiedMode 非Linux平台始终返回false
func IsCgroup2UnifiedMode() bool {
	return false
}
