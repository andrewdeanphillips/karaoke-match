package spotify

import "context"

// contextKey is an unexported type for context values defined by this
// package — guarding against collisions with keys other packages might add
// to the same context, per the standard library's context.WithValue advice.
type contextKey int

const accessTokenKey contextKey = iota

// ContextWithAccessToken returns a copy of ctx carrying the given Spotify
// access token, retrievable later with AccessTokenFromContext. The session
// middleware uses this to thread a visitor's resolved token from the cookie
// down to whatever calls the Spotify API on their behalf.
func ContextWithAccessToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, accessTokenKey, token)
}

// AccessTokenFromContext retrieves a Spotify access token previously stored
// with ContextWithAccessToken, and reports whether one was present.
func AccessTokenFromContext(ctx context.Context) (string, bool) {
	token, ok := ctx.Value(accessTokenKey).(string)
	return token, ok
}
