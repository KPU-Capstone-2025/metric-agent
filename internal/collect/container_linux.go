// internal/collect/container_linux.go
//go:build linux

package collect

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

type ContainerStat struct {
	ID         string
	Name       string
	CPUUsageNS uint64
	MemUsage   uint64
	NetRxBytes uint64
	NetTxBytes uint64
}

func ContainerUsage() ([]ContainerStat, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	defer cli.Close()

	containers, err := cli.ContainerList(context.Background(), container.ListOptions{})
	if err != nil {
		return nil, err
	}

	basePath := "/sys/fs/cgroup/system.slice"
	var stats []ContainerStat

	for _, c := range containers {
		name := strings.TrimPrefix(c.Names[0], "/")
		cgroupDir := "docker-" + c.ID + ".scope"

		// 1. CPU
		cpuPath := filepath.Join(basePath, cgroupDir, "cpu.stat")
		cpuData, _ := os.ReadFile(cpuPath)
		cpuNS := parseCgroupV2Cpu(string(cpuData))

		// 2. Memory
		memPath := filepath.Join(basePath, cgroupDir, "memory.current")
		memData, _ := os.ReadFile(memPath)
		memBytes, _ := strconv.ParseUint(strings.TrimSpace(string(memData)), 10, 64)

		// 3. Network (Docker Stats API)
		var rx, tx uint64
		s, err := cli.ContainerStats(context.Background(), c.ID, false)
		if err == nil {
			var v container.StatsResponse
			if err := json.NewDecoder(s.Body).Decode(&v); err == nil {
				for _, net := range v.Networks {
					rx += net.RxBytes
					tx += net.TxBytes
				}
			}
			s.Body.Close()
		}

		stats = append(stats, ContainerStat{
			ID:         c.ID[:12],
			Name:       name,
			CPUUsageNS: cpuNS,
			MemUsage:   memBytes,
			NetRxBytes: rx,
			NetTxBytes: tx,
		})
	}
	return stats, nil
}

func parseCgroupV2Cpu(data string) uint64 {
	lines := strings.Split(data, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "usage_usec") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				val, _ := strconv.ParseUint(parts[1], 10, 64)
				return val * 1000
			}
		}
	}
	return 0
}
