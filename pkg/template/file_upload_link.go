package template

import (
	"context"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/transport"
)

// UploadLink says whether the platform already holds a file bundle, and where
// to put it if not.
type UploadLink struct {
	// Present is true when the bundle is already cached, in which case URL is
	// empty and no upload is needed.
	Present bool

	// URL is a pre-signed destination for the bundle. It carries its own
	// authorisation.
	URL string
}

// FileUploadLink asks where to upload the file bundle identified by hash.
//
// Used by Build; call it directly only when driving the build sequence
// yourself.
//
// GET /templates/{templateID}/files/{hash}
func (s *Service) FileUploadLink(ctx context.Context, templateID, hash string) (*UploadLink, error) {
	resp, err := s.t.API().GetTemplatesTemplateIDFilesHashWithResponse(ctx, templateID, hash)
	if err != nil {
		return nil, err
	}
	link, err := transport.Parsed(resp.JSON201, resp.HTTPResponse, resp.Body)
	if err != nil {
		return nil, err
	}

	result := &UploadLink{Present: link.Present}
	if link.Url != nil {
		result.URL = *link.Url
	}
	return result, nil
}
