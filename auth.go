package main

import (
	"net/http"
)

func withBasicAuth(next interface{}) http.Handler {
	switch h := next.(type) {
	case http.Handler:
		return basicAuthHandler(h)
	case func(http.ResponseWriter, *http.Request):
		return basicAuthHandler(http.HandlerFunc(h))
	default:
		panic("[Auth] unsupported handler type")
	}
}

func basicAuthHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()

		requiredUsername := config.server.authUser
		requiredPassword := config.server.authPW

		if !ok || username != requiredUsername || password != requiredPassword {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}
