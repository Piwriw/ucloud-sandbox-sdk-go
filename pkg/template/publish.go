package template

import "context"

// Publish makes a template visible outside the team.
func (s *Service) Publish(ctx context.Context, templateID string) error {
	_, err := s.UpdateV2(ctx, templateID, true)
	return err
}

// Unpublish makes a template private to the team again.
func (s *Service) Unpublish(ctx context.Context, templateID string) error {
	_, err := s.UpdateV2(ctx, templateID, false)
	return err
}
