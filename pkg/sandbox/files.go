package sandbox

import (
	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/envd/filesystem"
)

// Filesystem reads and writes a sandbox's filesystem. Reach it through
// Sandbox.Files.
//
// Metadata operations -- listing, stat, rename, remove -- go over envd's
// connect-rpc service. File contents go over its HTTP /files endpoint, which
// streams rather than packing bytes into an RPC message.
type Filesystem struct {
	sbx *Sandbox
}

// entryTypeFrom converts a proto file type.
func entryTypeFrom(fileType filesystem.FileType) EntryType {
	switch fileType {
	case filesystem.FileType_FILE_TYPE_FILE:
		return EntryTypeFile
	case filesystem.FileType_FILE_TYPE_DIRECTORY:
		return EntryTypeDir
	case filesystem.FileType_FILE_TYPE_SYMLINK:
		return EntryTypeSymlink
	default:
		return EntryTypeUnknown
	}
}

// entryFrom converts a proto entry into this package's type.
func entryFrom(entry *filesystem.EntryInfo) *EntryInfo {
	if entry == nil {
		return nil
	}
	converted := &EntryInfo{
		Name:        entry.GetName(),
		Path:        entry.GetPath(),
		Type:        entryTypeFrom(entry.GetType()),
		Size:        entry.GetSize(),
		Mode:        entry.GetMode(),
		Permissions: entry.GetPermissions(),
		Owner:       entry.GetOwner(),
		Group:       entry.GetGroup(),
		Metadata:    entry.GetMetadata(),
	}
	if ts := entry.GetModifiedTime(); ts != nil {
		converted.ModifiedTime = ts.AsTime()
	}
	if target := entry.SymlinkTarget; target != nil {
		converted.SymlinkTarget = target
	}
	return converted
}

// eventTypeFrom converts a proto watch event type.
func eventTypeFrom(eventType filesystem.EventType) FilesystemEventType {
	switch eventType {
	case filesystem.EventType_EVENT_TYPE_CREATE:
		return EventTypeCreate
	case filesystem.EventType_EVENT_TYPE_WRITE:
		return EventTypeWrite
	case filesystem.EventType_EVENT_TYPE_REMOVE:
		return EventTypeRemove
	case filesystem.EventType_EVENT_TYPE_RENAME:
		return EventTypeRename
	case filesystem.EventType_EVENT_TYPE_CHMOD:
		return EventTypeChmod
	default:
		return ""
	}
}

// eventFrom converts a proto watch event.
func eventFrom(event *filesystem.FilesystemEvent) FilesystemEvent {
	return FilesystemEvent{
		Name:  event.GetName(),
		Type:  eventTypeFrom(event.GetType()),
		Entry: entryFrom(event.GetEntry()),
	}
}
