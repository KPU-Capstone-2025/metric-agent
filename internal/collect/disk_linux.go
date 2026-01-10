//go:build linux

package collect

import (
	"bufio"
	"os"
	"strings"
	"syscall"
)

type DiskStat struct {
	Mount      string
	TotalBytes uint64
	UsedBytes  uint64
	UsedPct    float64
}

func DiskUsage() ([]DiskStat, error) {
	mounts, err := readMountPoints()
	if err != nil {
		return nil, err
	}

	seen := map[string]bool{}
	out := make([]DiskStat, 0, len(mounts))

	for _, m := range mounts {
		if m == "" || seen[m] {
			continue
		}
		seen[m] = true

		var st syscall.Statfs_t
		if err := syscall.Statfs(m, &st); err != nil {
			continue
		}

		total := st.Blocks * uint64(st.Bsize)
		free := st.Bavail * uint64(st.Bsize)
		used := uint64(0)
		if total > free {
			used = total - free
		}
		pct := 0.0
		if total > 0 {
			pct = (float64(used) / float64(total)) * 100.0
		}
		out = append(out, DiskStat{Mount: m, TotalBytes: total, UsedBytes: used, UsedPct: pct})
	}

	return out, nil
}

func readMountPoints() ([]string, error) {
	f, err := os.Open("/proc/mounts")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var mounts []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		// device mount fstype options ...
		fields := strings.Fields(sc.Text())
		if len(fields) >= 2 {
			mounts = append(mounts, fields[1])
		}
	}
	return mounts, sc.Err()
}
