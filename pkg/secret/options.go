package secret

// CreateOptions are the optional arguments to Service.Create.
type CreateOptions struct {
	// Metadata is stored alongside the secret. The platform caps it at 32
	// entries, keys at 128 bytes, values at 1024 bytes, and the whole map at
	// 8192 bytes.
	Metadata map[string]string
}

// UpdateOptions are the optional arguments to Service.Update.
type UpdateOptions struct {
	// Metadata replaces whatever is stored on the secret. Leaving it nil keeps
	// the existing metadata; an empty non-nil map clears it.
	Metadata map[string]string
}

// ListOptions are the optional arguments to Service.List.
type ListOptions struct {
	// Limit is how many secrets to fetch per page. Zero lets the platform
	// choose; it caps the value at 100.
	Limit int
}
