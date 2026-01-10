//go:build linux

package collect

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

type NetStat struct {
	Iface   string
	RxBytes uint64
	TxBytes uint64
}

func NetBytes() ([]NetStat, error) {
	f, err := os.Open("/proc/net/dev")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var out []NetStat
	sc := bufio.NewScanner(f)
	lineNo := 0
	for sc.Scan() {
		lineNo++
		if lineNo <= 2 {
			continue
		}
		line := strings.TrimSpace(sc.Text())
		parts := strings.Split(line, ":")
		if len(parts) != 2 {
			continue
		}
		iface := strings.TrimSpace(parts[0])
		fields := strings.Fields(strings.TrimSpace(parts[1]))
		if len(fields) < 16 {
			continue
		}
		rx, _ := strconv.ParseUint(fields[0], 10, 64)
		tx, _ := strconv.ParseUint(fields[8], 10, 64)
		out = append(out, NetStat{Iface: iface, RxBytes: rx, TxBytes: tx})
	}
	return out, sc.Err()
}
