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

type clientIPKey struct{}

func WithClientIP(ctx context.Context, ip string) context.Context {
	return context.WithValue(ctx, clientIPKey{}, ip)
}

func ClientIPFromContext(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	ip, ok := ctx.Value(clientIPKey{}).(string)
	return ip, ok && ip != ""
}
