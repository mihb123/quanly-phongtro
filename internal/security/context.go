package security

import "context"

type authClaimsKey struct{}

func WithClaims(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, authClaimsKey{}, claims)
}

func ClaimsFromContext(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(authClaimsKey{}).(*Claims)
	return claims, ok
}
