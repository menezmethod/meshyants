//go:build !linux

package tier

func readSysTotalMemory() uint64 {
	return 0
}
