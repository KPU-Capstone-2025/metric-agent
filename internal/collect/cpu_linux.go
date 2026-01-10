//go:build linux

package collect

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type cpuTimes struct {
	user, nice, system, idle, iowait, irq, softirq, steal uint64
}

func readCPUTimes() (cpuTimes, error) {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return cpuTimes{}, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "cpu ") {
			fields := strings.Fields(line)
			if len(fields) < 8 {
				return cpuTimes{}, fmt.Errorf("unexpected /proc/stat cpu fields: %v", fields)
			}
			toU64 := func(s string) uint64 {
				v, _ := strconv.ParseUint(s, 10, 64)
				return v
			}
			return cpuTimes{
				user:    toU64(fields[1]),
				nice:    toU64(fields[2]),
				system:  toU64(fields[3]),
				idle:    toU64(fields[4]),
				iowait:  toU64(fields[5]),
				irq:     toU64(fields[6]),
				softirq: toU64(fields[7]),
				steal: func() uint64 {
					if len(fields) >= 9 {
						return toU64(fields[8])
					}
					return 0
				}(),
			}, nil
		}
	}
	if err := sc.Err(); err != nil {
		return cpuTimes{}, err
	}
	return cpuTimes{}, fmt.Errorf("cpu line not found")
}

func CPUUsage(sample time.Duration) (usagePct float64, iowaitPct float64, err error) {
	t1, err := readCPUTimes()
	if err != nil {
		return 0, 0, err
	}
	time.Sleep(sample)
	t2, err := readCPUTimes()
	if err != nil {
		return 0, 0, err
	}

	total1 := t1.user + t1.nice + t1.system + t1.idle + t1.iowait + t1.irq + t1.softirq + t1.steal
	total2 := t2.user + t2.nice + t2.system + t2.idle + t2.iowait + t2.irq + t2.softirq + t2.steal
	if total2 <= total1 {
		return 0, 0, fmt.Errorf("invalid cpu totals")
	}

	idle1 := t1.idle + t1.iowait
	idle2 := t2.idle + t2.iowait

	totald := float64(total2 - total1)
	idled := float64(idle2 - idle1)
	iowd := float64(t2.iowait - t1.iowait)

	usagePct = (1.0 - (idled / totald)) * 100.0
	iowaitPct = (iowd / totald) * 100.0
	return usagePct, iowaitPct, nil
}
