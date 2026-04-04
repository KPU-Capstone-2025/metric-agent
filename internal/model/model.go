package model

import "time"

type Payload struct { // 페이로드
	AgentVersion string    `json:"agentVersion"`
	Timestamp    time.Time `json:"timestamp"`
	IntervalSec  int       `json:"intervalSec"`
	Host         HostInfo  `json:"host"`
	Metrics      Metrics   `json:"metrics"`
}

type HostInfo struct { // 에이전트가 실행되고 있는 호스트 정보
	Hostname string `json:"hostname"`
	OS       string `json:"os"`
	Arch     string `json:"arch"`
}

type Metrics struct { // 에이전트에서 수집한 메트릭 정보
	CPU  CPUStat    `json:"cpu"`
	Mem  MemStat    `json:"mem"`
	Disk []DiskStat `json:"disk"`
	Net  []NetStat  `json:"net"`
}

type CPUStat struct { // CPU 사용량 통계
	UsagePct  float64 `json:"usagePct"`
	IowaitPct float64 `json:"iowaitPct"`
	Load1     float64 `json:"load1,omitempty"`
}

type MemStat struct { // 메모리 사용량 통계
	TotalBytes uint64  `json:"totalBytes"`
	UsedBytes  uint64  `json:"usedBytes"`
	UsedPct    float64 `json:"usedPct"`
}

type DiskStat struct { // 디스크 사용량 통계
	Mount      string  `json:"mount"`
	TotalBytes uint64  `json:"totalBytes"`
	UsedBytes  uint64  `json:"usedBytes"`
	UsedPct    float64 `json:"usedPct"`
}

type NetStat struct { // 네트워크 사용량 통계
	Iface   string `json:"iface"`
	RxBytes uint64 `json:"rxBytes"`
	TxBytes uint64 `json:"txBytes"`
}
