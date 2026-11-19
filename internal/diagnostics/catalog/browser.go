// Purpose: Own browser runtime, input, lifecycle and artifact failure messages.
// Pattern: Central diagnostic catalog.
// Usage: Browser provisioning and native tool adapters classify failures here.
// Related: internal/tools/browser, internal/tools/local/browser.go.
package catalog

const (
	BrowserInput       Message = "invalid browser request: %s"
	BrowserRuntime     Message = "browser runtime preparation failed: %w"
	BrowserUnsupported Message = "browser runtime is unavailable for %s; install Chromium and set MECA_BROWSER_EXECUTABLE to its absolute path"
	BrowserIntegrity   Message = "browser runtime checksum does not match the pinned release"
	BrowserArchive     Message = "browser archive contains an unsupported path or exceeds its extraction budget"
	BrowserDownload    Message = "browser runtime download returned HTTP %d"
	BrowserLimit       Message = "browser output exceeds its capture budget; narrow the inspected region"
	BrowserFailed      Message = "browser operation failed: %s; inspect current state before retrying an action"
	BrowserClosed      Message = "browser session is closed or has not been opened"
	BrowserStale       Message = "browser page changed after preparation; inspect the current page and prepare the action again"
	BrowserResult      Message = "browser result is unavailable or expired; inspect the current page again"
	BrowserScope       Message = "browser tools require system scope"
	BrowserTransfer    Message = "This interaction can send data or change remote state. Direct approval is required."
	BrowserInteraction Message = "Browser interaction may change application state"
	BrowserImage       Message = "browser screenshots require a vision-capable model; use snapshot or text inspection otherwise"
	BrowserClose       Message = "could not confirm browser shutdown: %w"
)

const (
	BrowserInvalidText       Message = "oversized or invalid text"
	BrowserInvalidSelector   Message = "selectors cannot be command options"
	BrowserInvalidValue      Message = "values cannot be command options"
	BrowserInvalidBounds     Message = "invalid result page bounds"
	BrowserInvalidResultRead Message = "result reads cannot execute actions"
	BrowserMissingResult     Message = "paging requires result_id and revision"
	BrowserMixedInput        Message = "choose an action or an inspection"
	BrowserUnknownView       Message = "unknown inspection"
	BrowserInvalidDepth      Message = "snapshot depth must be 1 to 12"
	BrowserInvalidURL        Message = "open requires an HTTP(S) URL without credentials"
	BrowserMissingTarget     Message = "action requires a selector or current element ref"
	BrowserMissingKey        Message = "press requires a key"
	BrowserInvalidDirection  Message = "invalid scroll direction"
	BrowserInvalidDistance   Message = "invalid scroll distance"
	BrowserInvalidViewport   Message = "viewport must be 320–2560 by 240–1600"
	BrowserUnknownAction     Message = "unknown browser action"
	BrowserUnexpectedURL     Message = "url is only accepted by open"
	BrowserInvalidRedirect   Message = "invalid download redirect"
	BrowserInvalidExecutable Message = "MECA_BROWSER_EXECUTABLE must name an executable absolute path"
	BrowserWrongTool         Message = "wrong browser tool"
	BrowserInvalidResponse   Message = "helper returned no complete JSON response"
	BrowserHelperExited      Message = "helper exited unsuccessfully"
	BrowserActionCompleted   Message = "Browser action completed; inspect the current page before the next dependent action."
	BrowserScreenshot        Message = "Current browser viewport screenshot"
)

const BrowserUnexpectedFields Message = "fields do not belong to the selected browser action or inspection"

const BrowserCapacityInvalid Message = "MECA_BROWSER_MAX_SESSIONS must be a positive integer"
const BrowserCapacityFull Message = "browser session capacity is occupied; close an unused browser before opening another"

const BrowserMemoryPressure Message = "system memory is under pressure; close unused browsers before opening a new session"
const BrowserMemoryObservation Message = "cannot check browser memory admission: %w"

const BrowserStableMetadata Message = "invalid Chrome Stable release metadata"
const BrowserStableVerification Message = "Chrome executable did not report the selected Stable version"

const BrowserInvalidCursor Message = "continue with the exact result_id, revision and next_offset from the latest page"

const BrowserInstallPreparing Message = "MECA 브라우저 도구를 준비하고 실행을 확인합니다."
const BrowserInstallReady Message = "브라우저 준비 완료 (페이지 열기·관찰·종료 확인)\n도우미: %s\n브라우저: %s"
const BrowserProbeFailed Message = "browser runtime check failed: %s"
const BrowserProbePage Message = "browser check did not reach its blank page"
const BrowserLibraryMissing Message = "browser runtime requires the Linux shared library %s; install the distribution package providing this library and run meca install --ext browser again"
const BrowserProcessFailed Message = "browser helper failed before returning a result: %s"
const BrowserRuntimeCleanupFailed Event = "browser_runtime_cleanup_failed"
