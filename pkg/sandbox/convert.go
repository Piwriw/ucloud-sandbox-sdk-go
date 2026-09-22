package sandbox

import (
	"time"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"
)

// Conversions from the generated types to this package's, so the public API
// does not expose the generator's spelling or its pointer-per-optional shape.

func infoFromDetail(d api.SandboxDetail) *Info {
	info := &Info{
		SandboxID:           d.SandboxID,
		TemplateID:          d.TemplateID,
		Name:                valueOr(d.Alias, ""),
		State:               d.State,
		CPUCount:            int(d.CpuCount),
		MemoryMB:            int(d.MemoryMB),
		DiskSizeMB:          int(d.DiskSizeMB),
		StartedAt:           d.StartedAt,
		EndAt:               d.EndAt,
		EnvdVersion:         d.EnvdVersion,
		Domain:              valueOr(d.Domain, ""),
		AllowInternetAccess: d.AllowInternetAccess,
		Network:             networkFrom(d.Network),
	}
	if d.Metadata != nil {
		info.Metadata = *d.Metadata
	}
	if d.VolumeMounts != nil {
		info.VolumeMounts = *d.VolumeMounts
	}
	return info
}

func infoFromListed(l api.ListedSandbox) Info {
	info := Info{
		SandboxID:   l.SandboxID,
		TemplateID:  l.TemplateID,
		Name:        valueOr(l.Alias, ""),
		State:       l.State,
		CPUCount:    int(l.CpuCount),
		MemoryMB:    int(l.MemoryMB),
		DiskSizeMB:  int(l.DiskSizeMB),
		StartedAt:   l.StartedAt,
		EndAt:       l.EndAt,
		EnvdVersion: l.EnvdVersion,
	}
	if l.Metadata != nil {
		info.Metadata = *l.Metadata
	}
	if l.VolumeMounts != nil {
		info.VolumeMounts = *l.VolumeMounts
	}
	return info
}

func metricsFrom(m api.SandboxMetric) Metrics {
	return Metrics{
		Timestamp:  time.Unix(m.TimestampUnix, 0).UTC(),
		CPUCount:   int(m.CpuCount),
		CPUUsedPct: float64(m.CpuUsedPct),
		MemTotal:   m.MemTotal,
		MemUsed:    m.MemUsed,
		DiskTotal:  m.DiskTotal,
		DiskUsed:   m.DiskUsed,
	}
}

func snapshotFrom(s api.SnapshotInfo) SnapshotInfo {
	return SnapshotInfo{SnapshotID: s.SnapshotID, Names: s.Names}
}
