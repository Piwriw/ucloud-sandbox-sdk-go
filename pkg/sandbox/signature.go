package sandbox

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// DownloadURL returns a URL that reads a file out of the sandbox.
//
// The URL carries its own signature, so it can be handed to something that has
// no API key -- a browser, or a service that only speaks HTTP.
func (s *Sandbox) DownloadURL(path string, opts FileURLOptions) string {
	return s.signedFileURL(path, "read", opts)
}

// UploadURL returns a URL that writes a file into the sandbox. The same caveats
// as DownloadURL apply, and rather more sharply: anyone holding it can write.
func (s *Sandbox) UploadURL(path string, opts FileURLOptions) string {
	return s.signedFileURL(path, "write", opts)
}

// signedFileURL builds a signed /files URL for one operation.
func (s *Sandbox) signedFileURL(path, operation string, opts FileURLOptions) string {
	user := s.resolveUser(opts.User)

	params := url.Values{"path": []string{path}}
	if user != "" {
		params.Set("username", user)
	}

	// Without an access token there is nothing to sign with, and the URL is
	// only usable by a caller that authenticates some other way.
	if s.conn.headers[headerAccessToken] != "" {
		signature, expiration := sign(path, operation, user,
			s.conn.headers[headerAccessToken], opts.ExpirationSeconds)
		params.Set("signature", signature)
		if expiration != 0 {
			params.Set("signature_expiration", strconv.FormatInt(expiration, 10))
		}
	}

	return s.conn.baseURL + "/files?" + params.Encode()
}

// sign computes the signature envd checks on a file URL, and the Unix time it
// expires at -- zero when it does not expire.
//
// The scheme is envd's: SHA-256 over the fields joined by colons, base64
// without padding, prefixed with the version.
func sign(path, operation, user, accessToken string, expirationSeconds int) (string, int64) {
	raw := fmt.Sprintf("%s:%s:%s:%s", path, operation, user, accessToken)

	var expiration int64
	if expirationSeconds > 0 {
		expiration = time.Now().Unix() + int64(expirationSeconds)
		raw = fmt.Sprintf("%s:%d", raw, expiration)
	}

	sum := sha256.Sum256([]byte(raw))
	encoded := strings.TrimRight(base64.StdEncoding.EncodeToString(sum[:]), "=")
	return "v1_" + encoded, expiration
}
