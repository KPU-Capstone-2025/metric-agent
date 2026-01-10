//go:build linux

package collect

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

func MemUsage() (totalBytes, usedBytes uint64, usedPct float64, err error) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0, 0, err
	}
	defer f.Close()

	var memTotalKB, memAvailKB uint64
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "MemTotal:") {
			memTotalKB = parseKB(line)
		} else if strings.HasPrefix(line, "MemAvailable:") {
			memAvailKB = parseKB(line)
		}
	}
	if err := sc.Err(); err != nil {
		return 0, 0, 0, err
	}
	if memTotalKB == 0 {
		return 0, 0, 0, nil
	}

	totalBytes = memTotalKB * 1024
	availBytes := memAvailKB * 1024
	if totalBytes > availBytes {
		usedBytes = totalBytes - availBytes
	}
	if totalBytes > 0 {
		usedPct = (float64(usedBytes) / float64(totalBytes)) * 100.0
	}
	return totalBytes, usedBytes, usedPct, nil
}

func parseKB(line string) uint64 {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return 0
	}
	v, _ := strconv.ParseUint(fields[1], 10, 64)
	return v
}
