// Purpose: Define operator and model-facing WebFetch diagnostics.
// Pattern: Central message catalog.
// Usage: The WebFetch transport, extraction and tool adapters use these messages.
// Related: internal/tools/browser/webfetch/client.go, internal/tools/browser/webfetch/documents.go.
package catalog

const (
	WebFetchTLSUnavailable Message = "web_fetch offline certificate verification is unavailable"
	WebFetchTLSUntrusted   Message = "web_fetch HTTPS requires a complete certificate chain trusted by this host"
	WebFetchDisabled       Message = "web_fetch is unavailable for this request"
	WebFetchURL            Message = "web_fetch requires a valid HTTP(S) URL without credentials"
	WebFetchBlocked        Message = "web_fetch target is blocked by the document's network policy"
	WebFetchPrivateURL     Message = "web_fetch requires the user to provide the exact URL for a local or private address"
	WebFetchMixedAddresses Message = "web_fetch refused a hostname that resolves to both public and private addresses"
	WebFetchRedirect       Message = "web_fetch exceeded the redirect limit"
	WebFetchTransfer       Message = "web_fetch could not read the requested URL: %w"
	WebFetchStatus         Message = "web_fetch origin returned HTTP %d"
	WebFetchMedia          Message = "web_fetch supports HTML, plain text and Markdown documents"
	WebFetchBodyLimit      Message = "web_fetch document exceeds its byte or markup budget"
	WebFetchEmpty          Message = "web_fetch found no readable document text"
	WebFetchExtraction     Message = "web_fetch could not extract a supported text document"
	WebFetchInput          Message = "provide url or result_id, revision and the current next_offset as offset"
	WebFetchCursor         Message = "web_fetch cursor is exhausted, stale or outside this request"
	WebFetchPending        Message = "web_fetch has too many unread documents; finish an existing document first"
	WebFetchContext        Message = "web_fetch has insufficient context for this page; use one page call at a time after context compaction"
	WebFetchReading        Message = "Reading web document"
	WebFetchBrowser        Message = "web_fetch needs an installed Chrome or Chromium; install the browser extension or configure MECA_BROWSER_EXECUTABLE"
	WebFetchBrowserStart   Message = "web_fetch could not start its isolated Chromium renderer"
	WebFetchBrowserPipe    Message = "web_fetch renderer connection ended: %w"
	WebFetchBrowserCDP     Message = "web_fetch renderer rejected %s"
	WebFetchNavigation     Message = "web_fetch renderer could not load the approved document"
)
