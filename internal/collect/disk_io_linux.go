//go:build linux

package collect

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

type DiskIOStat struct {
	ReadBytes  uint64
	WriteBytes uint64
}

// DiskIO returns cumulative total read/write bytes across all whole block devices.
// Uses /sys/block/ to distinguish whole disks from partitions.
func DiskIO() (DiskIOStat, error) {
	blockDevs := listBlockDevices()

	f, err := os.Open("/proc/diskstats")
	if err != nil {
		return DiskIOStat{}, err
	}
	defer f.Close()

	var totalRead, totalWrite uint64
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 14 {
			continue
		}
		name := fields[2]
		if len(blockDevs) > 0 && !blockDevs[name] {
			continue
		}
		// fallback filter when /sys/block is unavailable
		if strings.HasPrefix(name, "loop") || strings.HasPrefix(name, "ram") {
			continue
		}
		// /proc/diskstats field indices (0-based):
		// 3: reads completed, 5: sectors read, 7: writes completed, 9: sectors written
		// 1 sector = 512 bytes
		sectorsRead, _ := strconv.ParseUint(fields[5], 10, 64)
		sectorsWritten, _ := strconv.ParseUint(fields[9], 10, 64)
		totalRead += sectorsRead * 512
		totalWrite += sectorsWritten * 512
	}
	return DiskIOStat{ReadBytes: totalRead, WriteBytes: totalWrite}, sc.Err()
}

func listBlockDevices() map[string]bool {
	dir, err := os.Open("/sys/block")
	if err != nil {
		return nil
	}
	defer dir.Close()
	entries, _ := dir.Readdirnames(-1)
	m := make(map[string]bool, len(entries))
	for _, e := range entries {
		m[e] = true
	}
	return m
}
