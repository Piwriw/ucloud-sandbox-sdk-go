package transport

import (
	"context"
	"net/http"
)

// NextTokenHeader carries the cursor for the next page. The list endpoints
// return their items as a bare JSON array and put the cursor here rather than
// in the body.
const NextTokenHeader = "X-Next-Token"

// FetchPageFunc retrieves one page. It receives the cursor for the page to
// fetch — empty for the first — and returns the page's items along with the
// cursor for the page after it. An empty cursor means this was the last page.
type FetchPageFunc[T any] func(ctx context.Context, token string) ([]T, string, error)

// Paginator walks a paged listing one page at a time. Create one with
// NewPaginator; a zero Paginator is not usable.
//
// Nothing is fetched until the first NextItems or All, so building a Paginator
// cannot fail and takes no context.
type Paginator[T any] struct {
	fetch     FetchPageFunc[T]
	nextToken string
	hasNext   bool
}

// NewPaginator returns a Paginator that pages through fetch.
func NewPaginator[T any](fetch FetchPageFunc[T]) *Paginator[T] {
	return &Paginator[T]{fetch: fetch, hasNext: true}
}

// HasNext reports whether a further page may be available. It is true before
// the first fetch and stays true until a page comes back without a cursor.
func (p *Paginator[T]) HasNext() bool { return p.hasNext }

// NextItems fetches the next page. It returns nil once the listing is
// exhausted. On error the paginator is left where it was, so the call can be
// retried.
func (p *Paginator[T]) NextItems(ctx context.Context) ([]T, error) {
	if !p.hasNext {
		return nil, nil
	}
	items, nextToken, err := p.fetch(ctx, p.nextToken)
	if err != nil {
		return nil, err
	}
	p.nextToken = nextToken
	p.hasNext = nextToken != ""
	return items, nil
}

// All walks every remaining page and returns the items together. On error it
// returns what it had gathered so far alongside the error.
func (p *Paginator[T]) All(ctx context.Context) ([]T, error) {
	var all []T
	for p.HasNext() {
		items, err := p.NextItems(ctx)
		if err != nil {
			return all, err
		}
		all = append(all, items...)
	}
	return all, nil
}

// NextTokenFrom reads the pagination cursor out of a response's headers.
func NextTokenFrom(headers http.Header) string {
	if headers == nil {
		return ""
	}
	return headers.Get(NextTokenHeader)
}
