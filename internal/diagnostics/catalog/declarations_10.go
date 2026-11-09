// Purpose: Own product error and diagnostic message declarations.
// Pattern: Declarative diagnostic catalog.
// Usage: Product callers reference these templates through fault or registered diagnostics.
// Related: internal/diagnostics/catalog/messages.go, internal/diagnostics/fault/error.go.
package catalog

const (
	TransportProviderErrorBodyExceedsMiB                Message = "provider error body exceeds 16 MiB"
	TransportProviderMetadataExceedsByteLimit           Message = "provider metadata exceeds %d-byte limit"
	TransportProviderRequestBodyMustBeReplayable        Message = "provider request body must be replayable"
	TransportProviderSentNoResponseDataFor              Message = "provider sent no response data for %s"
	UpgradeArchiveChecksumIsMissing                     Message = "archive checksum is missing"
	UpgradeArchiveMustContainOneRegularBoundedMeca      Message = "archive must contain one regular bounded meca executable"
	UpgradeExpandedReleaseArchiveExceedsMiB             Message = "expanded release archive exceeds 512 MiB"
	UpgradeGitHubCredentialMustBeOneASCIIToken          Message = "GitHub credential must be one ASCII token"
	UpgradeGitHubReleaseRequestFailedAfterAttemptS      Message = "GitHub release request failed after %d attempt(s): %w"
	UpgradeGitHubReturnedHTTP                           Message = "GitHub returned HTTP %d"
	UpgradeInvalidGitHubReleaseEndpoint                 Message = "invalid GitHub release endpoint"
	UpgradeInvalidOrDuplicateArchiveChecksum            Message = "invalid or duplicate archive checksum"
	UpgradeInvalidReleaseDate                           Message = "invalid release date"
	UpgradeLatestReleaseIsNotASupportedPublished        Message = "latest release is not a supported published version"
	UpgradeMECAExecutableExceedsMiB                     Message = "MECA executable exceeds 256 MiB"
	UpgradeReadInstalledMECABinaryRunMecaInstall        Message = "read installed MECA binary: %w; run meca install before upgrade"
	UpgradeReleaseAccessFailedHTTPCheckRepositoryAccess Message = "release access failed (HTTP %d): check repository access and GH_TOKEN, GITHUB_TOKEN or gh auth login"
	UpgradeReleaseArchiveContainsNoMecaExecutable       Message = "release archive contains no meca executable"
	UpgradeReleaseArchiveSHADoesNotMatchChecksums       Message = "release archive SHA-256 does not match checksums.txt"
	UpgradeReleaseAssetIsMissingOrAmbiguous             Message = "release asset %q is missing or ambiguous"
	UpgradeReleaseDownloadRequiresHTTPS                 Message = "release download requires HTTPS"
	UpgradeReleaseInstallationFailed                    Message = "release %s installation failed: %w: %s"
	UpgradeReleaseResponseExceedsItsSizeLimit           Message = "release response exceeds its size limit"
	UpgradeTooManyReleaseDownloadRedirects              Message = "too many release download redirects"
	UpgradeUnsupportedReleasePlatform                   Message = "unsupported release platform"
	UpgradeUnsupportedReleaseVersion                    Message = "unsupported release version"
	UpgradeUpgradeRequiresAnInstalledRegularMECABinary  Message = "upgrade requires an installed regular MECA binary; run meca install first"
	WorkInvalidTerminalReason                           Message = "invalid terminal reason"
	WorkInvalidWorkTerminalStatus                       Message = "invalid work terminal status"
	WorkspaceInspectTheCurrentTargetAndResubmitThe      Message = "Inspect the current target and resubmit the needed operation."
	WorkspaceTheFileTargetChangedWhileSharedAccess      Message = "The file target changed while shared access was being prepared; nothing executed."
)
