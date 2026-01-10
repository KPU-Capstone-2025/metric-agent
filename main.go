package main

import (
	"encoding/json"
	"log"
	"os"
	"runtime"
	"time"

	"metric-agent/internal/collect"
	"metric-agent/internal/model"
)

func main() {
	interval := 10 * time.Second
	hostname, _ := os.Hostname()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Println("metric-agent started")

	for {
		start := time.Now()

		cpuUsage, iowait, err := collect.CPUUsage(500 * time.Millisecond)
		if err != nil {
			log.Printf("cpu error: %v", err)
		}

		totalMem, usedMem, usedPct, err := collect.MemUsage()
		if err != nil {
			log.Printf("mem error: %v", err)
		}

		disks, _ := collect.DiskUsage()
		nets, _ := collect.NetBytes()

		var diskStats []model.DiskStat
		for _, d := range disks {
			diskStats = append(diskStats, model.DiskStat{
				Mount:      d.Mount,
				TotalBytes: d.TotalBytes,
				UsedBytes:  d.UsedBytes,
				UsedPct:    d.UsedPct,
			})
		}

		var netStats []model.NetStat
		for _, n := range nets {
			netStats = append(netStats, model.NetStat{
				Iface:   n.Iface,
				RxBytes: n.RxBytes,
				TxBytes: n.TxBytes,
			})
		}

		payload := model.Payload{
			AgentVersion: "0.1.0",
			Timestamp:    time.Now().UTC(),
			IntervalSec:  int(interval.Seconds()),
			Host: model.HostInfo{
				Hostname: hostname,
				OS:       runtime.GOOS,
				Arch:     runtime.GOARCH,
			},
			Metrics: model.Metrics{
				CPU: model.CPUStat{
					UsagePct:  cpuUsage,
					IowaitPct: iowait,
				},
				Mem: model.MemStat{
					TotalBytes: totalMem,
					UsedBytes:  usedMem,
					UsedPct:    usedPct,
				},
				Disk: diskStats,
				Net:  netStats,
			},
		}

		b, _ := json.MarshalIndent(payload, "", "  ")
		log.Printf("collected (took %s)\n%s", time.Since(start), string(b))

		<-ticker.C
	}
}
