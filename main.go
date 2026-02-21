package main

import (
	"context"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/hpcloud/tail"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/metric"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"

	"metric-agent/internal/collect"
)

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func main() {
	ctx := context.Background()
	interval := 10 * time.Second

	monitoringID := getEnv("MONITORING_ID", "default-client")
	collectorURL := getEnv("COLLECTOR_URL", "localhost:4318")
	
	hostname, _ := os.Hostname()
	fullServerIdentity := monitoringID + "-" + hostname

	log.Printf("Agent started for [%s]. Data sending to [%s]", fullServerIdentity, collectorURL)

	headers := map[string]string{
		"X-Server-Group": monitoringID,
	}

	metricExporter, _ := otlpmetrichttp.New(ctx,
		otlpmetrichttp.WithEndpoint(collectorURL),
		otlpmetrichttp.WithInsecure(),
		otlpmetrichttp.WithHeaders(headers),
	)
	logExporter, _ := otlploghttp.New(ctx,
		otlploghttp.WithEndpoint(collectorURL),
		otlploghttp.WithInsecure(),
		otlploghttp.WithHeaders(headers),
	)

	res, _ := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String("metric-agent"),
			attribute.String("company.id", monitoringID),
			attribute.String("host.name", hostname),
		),
	)

	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter, sdkmetric.WithInterval(interval))),
		sdkmetric.WithResource(res),
	)
	defer meterProvider.Shutdown(ctx)
	otel.SetMeterProvider(meterProvider)

	logProvider := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(logExporter)),
		sdklog.WithResource(res),
	)
	defer logProvider.Shutdown(ctx)
	global.SetLoggerProvider(logProvider)

	go watchHostLogs(ctx)
	go watchContainerLogs(ctx)

	meter := otel.Meter("metric-agent")
	cpuGauge, _ := meter.Float64Gauge("system.cpu.usage")
	ramGauge, _ := meter.Float64Gauge("system.memory.usage")
	diskGauge, _ := meter.Float64Gauge("system.disk.usage")
	netRxGauge, _ := meter.Int64Gauge("system.network.rx_bytes")
	netTxGauge, _ := meter.Int64Gauge("system.network.tx_bytes")

	cCpuGauge, _ := meter.Int64Gauge("container.cpu.usage_ns")
	cMemGauge, _ := meter.Int64Gauge("container.memory.usage_bytes")
	cNetRxGauge, _ := meter.Int64Gauge("container.network.rx_bytes")
	cNetTxGauge, _ := meter.Int64Gauge("container.network.tx_bytes")

	ticker := time.NewTicker(interval)
	for range ticker.C {
		cpuUsage, _, _ := collect.CPUUsage(500 * time.Millisecond)
		_, _, ramPct, _ := collect.MemUsage()
		cpuGauge.Record(ctx, cpuUsage)
		ramGauge.Record(ctx, ramPct)

		disks, _ := collect.DiskUsage()
		for _, d := range disks {
			if d.Mount == "/" {
				diskGauge.Record(ctx, d.UsedPct)
				break
			}
		}

		nets, _ := collect.NetBytes()
		for _, n := range nets {
			if n.Iface == "eth0" || n.Iface == "ens5" || n.Iface == "enp1s0" {
				attrs := metric.WithAttributes(attribute.String("interface", n.Iface))
				netRxGauge.Record(ctx, int64(n.RxBytes), attrs)
				netTxGauge.Record(ctx, int64(n.TxBytes), attrs)
				break
			}
		}

		containers, err := collect.ContainerUsage()
		if err == nil {
			for _, c := range containers {
				attrs := metric.WithAttributes(
					attribute.String("container.id", c.ID),
					attribute.String("container.name", c.Name),
				)
				cCpuGauge.Record(ctx, int64(c.CPUUsageNS), attrs)
				cMemGauge.Record(ctx, int64(c.MemUsage), attrs)
				cNetRxGauge.Record(ctx, int64(c.NetRxBytes), attrs)
				cNetTxGauge.Record(ctx, int64(c.NetTxBytes), attrs)
			}
		}
	}
}

func watchHostLogs(ctx context.Context) {
	logger := global.GetLoggerProvider().Logger("host-logger")
	logPath := "/var/log/syslog"
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		logPath = "/var/log/messages"
	}

	t, err := tail.TailFile(logPath, tail.Config{Follow: true, ReOpen: true})
	if err != nil {
		return
	}

	for line := range t.Lines {
		now := time.Now()
		var r otellog.Record
		r.SetTimestamp(now)
		r.SetObservedTimestamp(now)
		r.SetBody(otellog.StringValue(line.Text))
		r.SetSeverityText("INFO")
		r.SetSeverity(otellog.SeverityInfo)
		logger.Emit(ctx, r)
	}
}

func watchContainerLogs(ctx context.Context) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return
	}

	containers, _ := cli.ContainerList(ctx, container.ListOptions{})
	for _, c := range containers {
		name := strings.TrimPrefix(c.Names[0], "/")
		if strings.Contains(name, "otel-collector") || strings.Contains(name, "metric-agent") {
			continue
		}

		go func(id string, cName string) {
			logger := global.GetLoggerProvider().Logger("container-logger")
			options := container.LogsOptions{ShowStdout: true, ShowStderr: true, Follow: true, Tail: "0"}
			reader, err := cli.ContainerLogs(ctx, id, options)
			if err != nil {
				return
			}
			defer reader.Close()

			header := make([]byte, 8)
			for {
				_, err := io.ReadFull(reader, header)
				if err != nil {
					break
				}
				count := uint32(header[4])<<24 | uint32(header[5])<<16 | uint32(header[6])<<8 | uint32(header[7])
				payload := make([]byte, count)
				_, err = io.ReadFull(reader, payload)
				if err != nil {
					break
				}

				msg := strings.TrimSpace(string(payload))
				if msg == "" {
					continue
				}

				now := time.Now()
				var r otellog.Record
				r.SetTimestamp(now)
				r.SetObservedTimestamp(now)
				r.AddAttributes(
					otellog.String("container.id", id[:12]),
					otellog.String("container.name", cName),
				)
				r.SetBody(otellog.StringValue(msg))
				r.SetSeverityText("INFO")
				r.SetSeverity(otellog.SeverityInfo)
				logger.Emit(ctx, r)
			}
		}(c.ID, name)
	}
}
