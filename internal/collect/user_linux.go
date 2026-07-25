//go:build linux

package collect

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type UserStat struct {
	Username string
	CPUPct   float64
	MemBytes uint64
}

// LoggedInUsers returns the number of currently logged-in users via the `who` command.
func LoggedInUsers() int {
	out, err := exec.Command("who").Output()
	if err != nil {
		return 0
	}
	count := 0
	for _, line := range strings.Split(string(out), "\n") {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count
}

// UserUsage returns per-user CPU% and memory (RSS bytes) for real login accounts.
// sampleDur controls the CPU measurement window.
func UserUsage(sampleDur time.Duration) ([]UserStat, error) {
	uidToUser := readPasswdLoginUsers()

	cpu1, err := readCPUTimeByUID()
	if err != nil {
		return nil, err
	}
	total1, err := readTotalJiffies()
	if err != nil {
		return nil, err
	}

	time.Sleep(sampleDur)

	cpu2, err := readCPUTimeByUID()
	if err != nil {
		return nil, err
	}
	total2, err := readTotalJiffies()
	if err != nil {
		return nil, err
	}

	mem, _ := readMemByUID()

	totalDelta := float64(total2 - total1)

	// Collect all UIDs that appear in any sample
	seen := map[uint64]struct{}{}
	for uid := range cpu1 {
		seen[uid] = struct{}{}
	}
	for uid := range cpu2 {
		seen[uid] = struct{}{}
	}
	for uid := range mem {
		seen[uid] = struct{}{}
	}

	var result []UserStat
	for uid := range seen {
		username, ok := uidToUser[uid]
		if !ok {
			continue
		}
		cpuPct := 0.0
		if totalDelta > 0 {
			delta := float64(cpu2[uid]) - float64(cpu1[uid])
			if delta < 0 {
				delta = 0
			}
			cpuPct = (delta / totalDelta) * 100.0
		}
		memBytes := mem[uid]
		if cpuPct == 0 && memBytes == 0 {
			continue
		}
		result = append(result, UserStat{
			Username: username,
			CPUPct:   cpuPct,
			MemBytes: memBytes,
		})
	}
	return result, nil
}

// readCPUTimeByUID aggregates (utime + stime) jiffies per UID from /proc/[pid]/stat.
func readCPUTimeByUID() (map[uint64]uint64, error) {
	dir, err := os.Open("/proc")
	if err != nil {
		return nil, err
	}
	defer dir.Close()

	entries, _ := dir.Readdirnames(-1)
	result := make(map[uint64]uint64)

	for _, entry := range entries {
		if _, err := strconv.ParseUint(entry, 10, 64); err != nil {
			continue
		}
		base := "/proc/" + entry
		uid, err := readUID(base + "/status")
		if err != nil {
			continue
		}
		cpuTime, err := readPidCPUTime(base + "/stat")
		if err != nil {
			continue
		}
		result[uid] += cpuTime
	}
	return result, nil
}

// readMemByUID aggregates VmRSS (bytes) per UID from /proc/[pid]/status.
func readMemByUID() (map[uint64]uint64, error) {
	dir, err := os.Open("/proc")
	if err != nil {
		return nil, err
	}
	defer dir.Close()

	entries, _ := dir.Readdirnames(-1)
	result := make(map[uint64]uint64)

	for _, entry := range entries {
		if _, err := strconv.ParseUint(entry, 10, 64); err != nil {
			continue
		}
		uid, vmRSS, err := readUIDAndMem("/proc/" + entry + "/status")
		if err != nil {
			continue
		}
		result[uid] += vmRSS * 1024 // kB → bytes
	}
	return result, nil
}

func readUID(statusPath string) (uint64, error) {
	f, err := os.Open(statusPath)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "Uid:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				return strconv.ParseUint(fields[1], 10, 64)
			}
		}
	}
	return 0, fmt.Errorf("uid not found in %s", statusPath)
}

func readUIDAndMem(statusPath string) (uid, vmRSSKB uint64, err error) {
	f, err := os.Open(statusPath)
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		switch {
		case strings.HasPrefix(line, "Uid:"):
			if fields := strings.Fields(line); len(fields) >= 2 {
				uid, _ = strconv.ParseUint(fields[1], 10, 64)
			}
		case strings.HasPrefix(line, "VmRSS:"):
			if fields := strings.Fields(line); len(fields) >= 2 {
				vmRSSKB, _ = strconv.ParseUint(fields[1], 10, 64)
			}
		}
	}
	return uid, vmRSSKB, nil
}

func readPidCPUTime(statPath string) (uint64, error) {
	data, err := os.ReadFile(statPath)
	if err != nil {
		return 0, err
	}
	// /proc/[pid]/stat: pid (comm) state ppid ... utime(14) stime(15) ...
	// The comm field may contain spaces and is wrapped in (), find the last ')' first.
	s := string(data)
	end := strings.LastIndex(s, ")")
	if end < 0 {
		return 0, fmt.Errorf("malformed stat")
	}
	fields := strings.Fields(s[end+1:])
	// After closing ')': state(0) ppid(1) pgrp(2) ... utime(11) stime(12)
	if len(fields) < 13 {
		return 0, fmt.Errorf("too few fields in stat")
	}
	utime, _ := strconv.ParseUint(fields[11], 10, 64)
	stime, _ := strconv.ParseUint(fields[12], 10, 64)
	return utime + stime, nil
}

func readTotalJiffies() (uint64, error) {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return 0, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "cpu ") {
			var total uint64
			for _, v := range strings.Fields(line)[1:] {
				n, _ := strconv.ParseUint(v, 10, 64)
				total += n
			}
			return total, nil
		}
	}
	return 0, fmt.Errorf("cpu line not found")
}

// readPasswdLoginUsers returns UID→username only for accounts with a real login shell.
func readPasswdLoginUsers() map[uint64]string {
	m := map[uint64]string{}
	f, err := os.Open("/etc/passwd")
	if err != nil {
		return m
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		parts := strings.Split(sc.Text(), ":")
		if len(parts) < 7 {
			continue
		}
		shell := parts[6]
		if strings.Contains(shell, "nologin") ||
			strings.Contains(shell, "/bin/false") ||
			strings.Contains(shell, "/sbin/false") ||
			strings.Contains(shell, "/usr/sbin/false") {
			continue
		}
		uid, err := strconv.ParseUint(parts[2], 10, 64)
		if err != nil {
			continue
		}
		m[uid] = parts[0]
	}
	return m
}
