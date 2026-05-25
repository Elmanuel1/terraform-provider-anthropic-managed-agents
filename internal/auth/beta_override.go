package auth

import (
	"context"
	"net/http"
)

type betaOverride struct {
	inner Credentials
	beta  string
}

// WithBeta wraps any Credentials and overrides the anthropic-beta header after
// the inner credential authenticates. Use this when an endpoint requires a
// different beta value than the credential's default (e.g. SkillsBeta over WIF).
func WithBeta(creds Credentials, beta string) Credentials {
	return betaOverride{inner: creds, beta: beta}
}

func (b betaOverride) Authenticate(ctx context.Context, req *http.Request) error {
	if err := b.inner.Authenticate(ctx, req); err != nil {
		return err
	}
	req.Header.Set(HeaderBeta, b.beta)
	return nil
}
