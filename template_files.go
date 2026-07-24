package sandbox

// Template COPY steps bundle local files from fileContextPath, upload them to the
// template (content-addressed by filesHash), then the build API applies COPY src→dest
// during the image build. See BuildTemplate: create template → upload files → trigger build.

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
)

// Copy adds a Dockerfile-like COPY step. Local sources under fileContextPath are
// hashed and uploaded before the build is triggered.
func (t *TemplateBuilder) Copy(src, dest string) *TemplateBuilder {
	t.addInstruction(Instruction{Type: InstructionCopy, Args: []string{src, dest}})
	return t
}

func (t *TemplateBuilder) prepareSteps() ([]Instruction, error) {
	steps := append([]Instruction(nil), t.instructions...)
	for i, inst := range steps {
		if inst.Type != InstructionCopy {
			continue
		}
		if len(inst.Args) < 2 {
			return nil, fmt.Errorf("COPY requires src and dest")
		}
		if t.fileContextPath == "" {
			return nil, fmt.Errorf("fileContextPath is required for COPY %q", inst.Args[0])
		}
		hash, err := calculateFilesHash(inst.Args[0], inst.Args[1], t.fileContextPath)
		if err != nil {
			return nil, err
		}
		steps[i].FilesHash = hash
	}
	return steps, nil
}

func validateRelativeCopyPath(src string) error {
	if filepath.IsAbs(src) {
		return fmt.Errorf("COPY source %q must be relative to fileContextPath", src)
	}
	clean := filepath.Clean(src)
	if clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("COPY source %q escapes fileContextPath", src)
	}
	return nil
}

func collectFileContextFiles(src, contextPath string) ([]string, error) {
	if err := validateRelativeCopyPath(src); err != nil {
		return nil, err
	}

	root := filepath.Join(contextPath, src)
	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("COPY source %q: %w", src, err)
	}

	var files []string
	if info.IsDir() {
		err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				rel, relErr := filepath.Rel(contextPath, path)
				if relErr != nil {
					return relErr
				}
				files = append(files, toPosix(rel))
				return nil
			}
			if !d.Type().IsRegular() {
				return nil
			}
			rel, relErr := filepath.Rel(contextPath, path)
			if relErr != nil {
				return relErr
			}
			files = append(files, toPosix(rel))
			return nil
		})
		if err != nil {
			return nil, err
		}
	} else {
		files = append(files, toPosix(src))
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no files found for COPY source %q", src)
	}
	sort.Strings(files)
	return files, nil
}

func calculateFilesHash(src, dest, contextPath string) (string, error) {
	files, err := collectFileContextFiles(src, contextPath)
	if err != nil {
		return "", err
	}

	h := sha256.New()
	h.Write(fmt.Appendf(nil, "COPY %s %s", src, dest))

	for _, rel := range files {
		path := filepath.Join(contextPath, filepath.FromSlash(rel))
		h.Write([]byte(rel))

		info, err := os.Lstat(path)
		if err != nil {
			return "", err
		}
		h.Write(fmt.Appendf(nil, "%o", info.Mode()))
		h.Write(fmt.Appendf(nil, "%d", info.Size()))

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

func tarGzFileContext(src, contextPath string) ([]byte, error) {
	files, err := collectFileContextFiles(src, contextPath)
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
			f, err := os.Open(path)
			if err != nil {
				return nil, err
			}
			_, copyErr := io.Copy(tw, f)
			closeErr := f.Close()
			if copyErr != nil {
				return nil, copyErr
			}
			if closeErr != nil {
				return nil, closeErr
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

func (c *Client) uploadTemplateFiles(ctx context.Context, templateID string, template *TemplateBuilder, steps []Instruction) error {
	uploaded := make(map[string]string)
	for _, step := range steps {
		if step.Type != InstructionCopy || step.FilesHash == "" {
			continue
		}
		if _, ok := uploaded[step.FilesHash]; ok {
			continue
		}

		path := fmt.Sprintf("/templates/%s/files/%s", templateID, step.FilesHash)
		var link templateFileUploadLink
		if err := c.doRequest(ctx, http.MethodGet, path, nil, &link); err != nil {
			return err
		}
		if link.Present {
			uploaded[step.FilesHash] = "present"
			continue
		}
		uploadURL := link.URL
		if uploadURL == "" {
			uploadURL = link.UploadURL
		}
		if uploadURL == "" {
			return fmt.Errorf("empty upload url for files hash %s", step.FilesHash)
		}

		payload, err := tarGzFileContext(step.Args[0], template.fileContextPath)
		if err != nil {
			return err
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPut, uploadURL, bytes.NewReader(payload))
		if err != nil {
			return err
		}

		httpClient := c.newSandboxHTTPClient()
		resp, err := httpClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
			return fmt.Errorf("upload template files: http %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
		}

		uploaded[step.FilesHash] = uploadURL
	}
	return nil
}

func toPosix(path string) string {
	return filepath.ToSlash(path)
}
