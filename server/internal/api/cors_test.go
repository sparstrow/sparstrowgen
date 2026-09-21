package api

import (
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

/* The browser asks permission before it sends anything but a plain GET or POST,
and it asks for ONE method at a time. Every Go test here calls the handler
directly, so none of them preflights — which is how a route was added with a
method the browser was never allowed to use, and it passed every test while
being impossible to call from the app (found 2026-09-21, uploading an avatar
with PUT).

So this asserts the rule rather than one route: whatever the router serves, the
preflight allows. Adding a route with a new method fails here until the CORS
list is updated, which is the moment to decide whether to widen it or to use a
method that is already allowed. */

func allowedMethods(t *testing.T, a *API) map[string]bool {
	t.Helper()
	req, err := http.NewRequest(http.MethodOptions, "/api/auth/session", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Origin", testOrigin)
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)

	rec := httptest.NewRecorder()
	a.Routes().ServeHTTP(rec, req)

	allowed := map[string]bool{}
	for _, m := range strings.Split(rec.Header().Get("Access-Control-Allow-Methods"), ",") {
		if m = strings.TrimSpace(m); m != "" {
			allowed[strings.ToUpper(m)] = true
		}
	}
	if len(allowed) == 0 {
		t.Fatal("the preflight allowed no methods at all")
	}
	return allowed
}

func TestEveryRouteUsesAMethodTheBrowserIsAllowedToSend(t *testing.T) {
	r := newRig(t)
	allowed := allowedMethods(t, r.api)

	// The daemon's own routes are not called by a browser and never preflight,
	// so they are not held to this.
	daemonOnly := func(route string) bool {
		return route == "/daemon" || strings.HasPrefix(route, "/daemon/")
	}

	offending := map[string][]string{}
	walk := func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		if daemonOnly(route) || allowed[strings.ToUpper(method)] {
			return nil
		}
		offending[method] = append(offending[method], route)
		return nil
	}
	if err := chi.Walk(r.api.Routes().(chi.Routes), walk); err != nil {
		t.Fatalf("walking the routes: %v", err)
	}

	if len(offending) > 0 {
		methods := make([]string, 0, len(offending))
		for m := range offending {
			methods = append(methods, m)
		}
		sort.Strings(methods)
		for _, m := range methods {
			sort.Strings(offending[m])
			t.Errorf("%s is served on %v but is not in Access-Control-Allow-Methods, so a browser cannot call it", m, offending[m])
		}
		t.Fatal("either use an allowed method or widen the CORS list deliberately")
	}
}
