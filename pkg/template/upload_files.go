package template

// A COPY step does not carry its files in the build request. Instead the local
// sources are hashed, the bundle is uploaded to a URL the platform hands out
// for that hash, and the step references the hash. The platform already holding
// a bundle is what makes a repeated build skip the upload.
//
// So Build runs: create template -> hash and upload each COPY bundle -> start
// build.

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/errdefs"
)

// prepareSteps returns the builder's steps with each COPY step's file hash
// filled in.
func (b *Builder) prepareSteps() ([]Instruction, error) {
	steps := b.steps()
	for i, inst := range steps {
		if inst.Type != InstructionCopy && inst.Type != InstructionAdd {
			continue
		}
		if len(inst.Args) < 2 {
			return nil, fmt.Errorf("template: COPY requires a source and a destination")
		}
		if b.fileContextPath == "" {
			return nil, fmt.Errorf(
				"template: COPY %q needs a file context; set BuilderOptions.FileContextPath",
				inst.Args[0])
		}
		hash, err := hashFileContext(inst.Args[0], inst.Args[1], b.fileContextPath)
		if err != nil {
			return nil, err
		}
		steps[i].FilesHash = hash
	}
	return steps, nil
}

// uploadFiles uploads the bundle behind every COPY step that the platform does
// not already have. Steps sharing a hash are uploaded once.
func (s *Service) uploadFiles(ctx context.Context, templateID string, b *Builder, steps []Instruction) error {
	seen := make(map[string]bool)

	for _, step := range steps {
		if step.Type != InstructionCopy || step.FilesHash == "" || seen[step.FilesHash] {
			continue
		}
		seen[step.FilesHash] = true

		link, err := s.FileUploadLink(ctx, templateID, step.FilesHash)
		if err != nil {
			return err
		}
		if link.Present {
			continue
		}
		if link.URL == "" {
			return &errdefs.FileUploadError{SandboxError: errdefs.SandboxError{
				Message: fmt.Sprintf("template: no upload URL for file bundle %s", step.FilesHash),
			}}
		}

		payload, err := tarGzFileContext(step.Args[0], b.fileContextPath)
		if err != nil {
			return err
		}
		if err := s.putBundle(ctx, link.URL, payload); err != nil {
			return err
		}
	}
	return nil
}

// putBundle uploads one bundle to a pre-signed URL.
//
// The URL carries its own authorisation, so this request deliberately does not
// go through the API client: sending the account's API key to whatever storage
// endpoint the platform names would disclose it needlessly.
func (s *Service) putBundle(ctx context.Context, uploadURL string, payload []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, uploadURL, bytes.NewReader(payload))
	if err != nil {
		return err
	}

	resp, err := s.t.HTTPClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return &errdefs.FileUploadError{SandboxError: errdefs.SandboxError{
			Message: fmt.Sprintf("template: upload file bundle: HTTP %d: %s",
				resp.StatusCode, strings.TrimSpace(string(body))),
		}}
	}
	return nil
}

// hashFileContext computes the cache key for one COPY step.
//
// The hash covers the source and destination paths, and every file's relative
// path, mode, size and contents. Anything that would change what the step
// produces therefore changes the hash, and anything that would not — the order
// the filesystem happens to report entries in — does not, because the list is
// sorted.
func hashFileContext(src, dest, contextPath string) (string, error) {
	files, err := collectFileContext(src, contextPath)
	if err != nil {
		return "", err
	}

	h := sha256.New()
	fmt.Fprintf(h, "COPY %s %s", src, dest)

	for _, rel := range files {
		path := filepath.Join(contextPath, filepath.FromSlash(rel))
		h.Write([]byte(rel))

		info, err := os.Lstat(path)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(h, "%o", info.Mode())
		fmt.Fprintf(h, "%d", info.Size())

		if info.Mode().IsRegular() {
			content, err := os.ReadFile(path)
			if err != nil {
				return "", err
			}
			h.Write(content)
		}
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// tarGzFileContext bundles a COPY step's sources into a gzipped tar.
func tarGzFileContext(src, contextPath string) ([]byte, error) {
	files, err := collectFileContext(src, contextPath)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	for _, rel := range files {
		path := filepath.Join(contextPath, filepath.FromSlash(rel))
		info, err := os.Lstat(path)
		if err != nil {
			return nil, err
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return nil, err
		}
		header.Name = rel
		if err := tw.WriteHeader(header); err != nil {
			return nil, err
		}

		if info.Mode().IsRegular() {
			if err := copyFileInto(tw, path); err != nil {
				return nil, err
			}
		}
	}

	if err := tw.Close(); err != nil {
		return nil, err
	}
	if err := gw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// copyFileInto streams one file into the tar writer.
func copyFileInto(tw *tar.Writer, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(tw, f)
	return err
}

// collectFileContext lists the paths a COPY source covers, relative to the
// context directory and sorted so the result is reproducible.
func collectFileContext(src, contextPath string) ([]string, error) {
	if err := checkRelativePath(src); err != nil {
		return nil, err
	}

	root := filepath.Join(contextPath, src)
	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("template: COPY source %q: %w", src, err)
	}

	if !info.IsDir() {
		return []string{filepath.ToSlash(src)}, nil
	}

	var files []string
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		// Directories are included so that empty ones survive the round trip;
		// anything that is neither a directory nor a regular file (a socket,
		// a device) has no meaning inside a template image.
		if !d.IsDir() && !d.Type().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(contextPath, path)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, err
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("template: no files found for COPY source %q", src)
	}
	sort.Strings(files)
	return files, nil
}

// checkRelativePath rejects a COPY source that is absolute or that climbs out
// of the file context, either of which would bundle files the caller did not
// mean to ship.
func checkRelativePath(src string) error {
	if filepath.IsAbs(src) {
		return fmt.Errorf("template: COPY source %q must be relative to the file context", src)
	}
	clean := filepath.Clean(src)
	if clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("template: COPY source %q escapes the file context", src)
	}
	return nil
}
