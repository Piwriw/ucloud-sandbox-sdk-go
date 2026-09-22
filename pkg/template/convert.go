package template

import "github.com/ucloud/ucloud-sandbox-sdk-go/pkg/api"

// Conversions from the generated types to this package's. They exist so that
// the public API does not expose the generator's spelling — CpuCount, Url — nor
// its pointer-per-optional-field shape.

func templateInfoFrom(t api.Template) Info {
	info := Info{
		TemplateID:    t.TemplateID,
		BuildID:       t.BuildID,
		Names:         t.Names,
		Aliases:       t.Aliases,
		Public:        t.Public,
		CPUCount:      int(t.CpuCount),
		MemoryMB:      int(t.MemoryMB),
		DiskSizeMB:    int(t.DiskSizeMB),
		BuildCount:    int(t.BuildCount),
		SpawnCount:    t.SpawnCount,
		EnvdVersion:   t.EnvdVersion,
		CreatedAt:     t.CreatedAt,
		UpdatedAt:     t.UpdatedAt,
		LastSpawnedAt: t.LastSpawnedAt,
	}
	if t.CreatedBy != nil {
		info.CreatedByID = t.CreatedBy.Id.String()
		if t.CreatedBy.Email != nil {
			info.CreatedByEmail = *t.CreatedBy.Email
		}
	}
	return info
}

func buildFrom(b api.TemplateBuild) Build {
	build := Build{
		BuildID:    b.BuildID.String(),
		Status:     b.Status,
		CPUCount:   int(b.CpuCount),
		MemoryMB:   int(b.MemoryMB),
		CreatedAt:  b.CreatedAt,
		UpdatedAt:  b.UpdatedAt,
		FinishedAt: b.FinishedAt,
	}
	if b.DiskSizeMB != nil {
		build.DiskSizeMB = int(*b.DiskSizeMB)
	}
	if b.EnvdVersion != nil {
		build.EnvdVersion = *b.EnvdVersion
	}
	return build
}

func withBuildsFrom(t api.TemplateWithBuilds, nextToken string) *WithBuilds {
	builds := make([]Build, 0, len(t.Builds))
	for _, b := range t.Builds {
		builds = append(builds, buildFrom(b))
	}
	return &WithBuilds{
		TemplateID:    t.TemplateID,
		Public:        t.Public,
		Names:         t.Names,
		Aliases:       t.Aliases,
		SpawnCount:    t.SpawnCount,
		CreatedAt:     t.CreatedAt,
		UpdatedAt:     t.UpdatedAt,
		LastSpawnedAt: t.LastSpawnedAt,
		Builds:        builds,
		NextToken:     nextToken,
	}
}

func tagFrom(t api.TemplateTag) Tag {
	return Tag{Tag: t.Tag, BuildID: t.BuildID.String(), CreatedAt: t.CreatedAt}
}

func statusFrom(info api.TemplateBuildInfo) *Status {
	status := &Status{
		TemplateID: info.TemplateID,
		BuildID:    info.BuildID,
		Status:     info.Status,
		Logs:       info.Logs,
		LogEntries: logEntriesFrom(info.LogEntries),
	}
	if info.Reason != nil {
		status.Reason = &StatusReason{Message: info.Reason.Message}
		if info.Reason.Step != nil {
			status.Reason.Step = *info.Reason.Step
		}
		if info.Reason.LogEntries != nil {
			status.Reason.LogEntries = logEntriesFrom(*info.Reason.LogEntries)
		}
	}
	return status
}
