package sandbox

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// AssignedTemplateTags describes tags assigned to a template build.
type AssignedTemplateTags struct {
	BuildID string   `json:"buildID"`
	Tags    []string `json:"tags"`
}

// TemplateTag describes a tag assigned to a template build.
type TemplateTag struct {
	Tag       string    `json:"tag"`
	BuildID   string    `json:"buildID"`
	CreatedAt time.Time `json:"createdAt"`
}

// AssignTemplateTags assigns tags to the build identified by target (name:tag).
func (c *Client) AssignTemplateTags(ctx context.Context, target string, tags []string) (*AssignedTemplateTags, error) {
	var result AssignedTemplateTags
	body := struct {
		Target string   `json:"target"`
		Tags   []string `json:"tags"`
	}{Target: target, Tags: tags}
	if err := c.doRequest(ctx, http.MethodPost, "/templates/tags", body, &result); err != nil {
		return nil, fmt.Errorf("assign template tags: %w", err)
	}
	return &result, nil
}

// DeleteTemplateTags deletes tags from a template.
func (c *Client) DeleteTemplateTags(ctx context.Context, name string, tags []string) error {
	body := struct {
		Name string   `json:"name"`
		Tags []string `json:"tags"`
	}{Name: name, Tags: tags}
	if err := c.doRequest(ctx, http.MethodDelete, "/templates/tags", body, nil); err != nil {
		return fmt.Errorf("delete template tags: %w", err)
	}
	return nil
}

// ListTemplateTags lists all tags assigned to a template.
func (c *Client) ListTemplateTags(ctx context.Context, templateID string) ([]TemplateTag, error) {
	var result []TemplateTag
	path := "/templates/" + url.PathEscape(templateID) + "/tags"
	if err := c.doRequest(ctx, http.MethodGet, path, nil, &result); err != nil {
		return nil, fmt.Errorf("list template tags: %w", err)
	}
	return result, nil
}
