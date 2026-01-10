package model

import "time"

type Payload struct {
	AgentVersion string    `json:"agentVersion"`
	Timestamp    time.Time `json:"timestamp"`
	IntervalSec  int       `json:"intervalSec"`
	Host         HostInfo  `json:"host"`
	Metrics      Metrics   `json:"metrics"`
}

type HostInfo struct {
	Hostname string `json:"hostname"`
	OS       string `json:"os"`
	Arch     string `json:"arch"`
}

type Metrics struct {
	CPU  CPUStat    `json:"cpu"`
	Mem  MemStat    `json:"mem"`
	Disk []DiskStat `json:"disk"`
	Net  []NetStat  `json:"net"`
}

type CPUStat struct {
	UsagePct  float64 `json:"usagePct"`
	IowaitPct float64 `json:"iowaitPct"`
	Load1     float64 `json:"load1,omitempty"`
}

type MemStat struct {
	TotalBytes uint64  `json:"totalBytes"`
	UsedBytes  uint64  `json:"usedBytes"`
	UsedPct    float64 `json:"usedPct"`
}

type DiskStat struct {
	Mount      string  `json:"mount"`
	TotalBytes uint64  `json:"totalBytes"`
	UsedBytes  uint64  `json:"usedBytes"`
	UsedPct    float64 `json:"usedPct"`
}

type NetStat struct {
	Iface   string `json:"iface"`
	RxBytes uint64 `json:"rxBytes"`
	TxBytes uint64 `json:"txBytes"`
}
