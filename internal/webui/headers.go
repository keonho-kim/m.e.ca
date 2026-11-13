// Purpose: Apply browser document policies without restricting native API clients.
// Pattern: Static UI response policy owned by the asset adapter.
// Usage: Handler applies these before cache negotiation, including 304 responses.
// Related: internal/webui/assets.go, webui/e2e/browser-policy.spec.ts.
package webui

import "net/http"

func browserHeaders(w http.ResponseWriter) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' blob: data:; font-src 'self'; connect-src 'self'; object-src 'none'; base-uri 'none'; frame-ancestors 'none'; form-action 'self'")
	// Observe compatibility before enforcement. No source samples or application
	// data are sent to a reporting service; browser tests inspect directive names.
	w.Header().Set("Content-Security-Policy-Report-Only", "require-trusted-types-for 'script'")
	w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=(), clipboard-read=(self), clipboard-write=(self)")
	w.Header().Set("Cross-Origin-Resource-Policy", "same-origin")
	w.Header().Set("Cross-Origin-Opener-Policy", "same-origin-allow-popups")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Referrer-Policy", "no-referrer")
}
