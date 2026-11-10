// Purpose: Select the requested distribution's latest platform archive and verify SHA-256 before extraction.
// Pattern: Release download use case.
// Usage: Run passes a private staging root to fetch.
// Related: internal/commands/upgrade/request.go, internal/commands/upgrade/archive.go, .goreleaser.yaml.
package upgrade

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"

	"io"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/distribution"
)

const maxArchiveBytes = 256 * 1024 * 1024

var releaseTag = regexp.MustCompile(`^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.([0-9]{8})$`)

type asset struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Size int64  `json:"size"`
}

type release struct {
	Version    string  `json:"tag_name"`
	Draft      bool    `json:"draft"`
	Prerelease bool    `json:"prerelease"`
	Assets     []asset `json:"assets"`
}

func (c *Client) fetch(ctx context.Context, directory *os.Root, kind distribution.Kind) (string, error) {
	if kind != distribution.TUI && kind != distribution.Full {
		return "", faults.Errorf(diagcatalog.UpgradeUnsupportedDistribution)
	}
	var latest release
	err := c.read(ctx, "/releases/latest", "application/vnd.github+json", func(reader io.Reader) error {
		body, err := readInfo(reader)
		if err == nil {
			var candidate release
			err = json.Unmarshal(body, &candidate)
			if err == nil {
				latest = candidate
			}
		}
		return err
	})
	if err != nil {
		return "", err
	}
	name, err := archiveName(latest.Version, kind, runtime.GOOS, runtime.GOARCH)
	if err != nil || latest.Draft || latest.Prerelease {
		return "", faults.Errorf(diagcatalog.UpgradeLatestReleaseIsNotASupportedPublished)
	}
	find := func(name string) (asset, error) {
		var matches []asset
		for _, candidate := range latest.Assets {
			if candidate.Name == name {
				matches = append(matches, candidate)
			}
		}
		if len(matches) != 1 || matches[0].ID <= 0 || matches[0].Size <= 0 {
			return asset{}, faults.Errorf(diagcatalog.UpgradeReleaseAssetIsMissingOrAmbiguous, name)
		}
		return matches[0], nil
	}
	archive, err := find(name)
	if err != nil {
		return "", err
	}
	checksums, err := find("checksums.txt")
	if err != nil {
		return "", err
	}
	if archive.Size > maxArchiveBytes || checksums.Size > maxInfoBytes {
		return "", errTooLarge
	}
	var expected string
	err = c.read(ctx, assetPath(checksums), "application/octet-stream", func(reader io.Reader) error {
		body, err := readInfo(reader)
		if err == nil {
			expected, err = archiveChecksum(string(body), name)
		}
		return err
	})
	if err != nil {
		return "", err
	}
	file, err := directory.OpenFile("release.tar.gz", os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	if err != nil {
		return "", err
	}
	hash := sha256.New()
	err = c.read(ctx, assetPath(archive), "application/octet-stream", func(reader io.Reader) error {
		if err := file.Truncate(0); err != nil {
			return err
		}
		if _, err := file.Seek(0, io.SeekStart); err != nil {
			return err
		}
		hash.Reset()
		count, err := io.Copy(io.MultiWriter(file, hash), io.LimitReader(reader, maxArchiveBytes+1))
		if count > maxArchiveBytes {
			return errTooLarge
		}
		if err == nil && count != archive.Size {
			err = io.ErrUnexpectedEOF
		}
		return err
	})
	err = errors.Join(err, file.Close())
	if err != nil {
		return "", err
	}
	if hex.EncodeToString(hash.Sum(nil)) != expected {
		return "", faults.Errorf(diagcatalog.UpgradeReleaseArchiveSHADoesNotMatchChecksums)
	}
	if err := extract(directory); err != nil {
		return "", err
	}
	return latest.Version, nil
}

func assetPath(asset asset) string {
	return "/releases/assets/" + strconv.FormatInt(asset.ID, 10)
}

func archiveName(version string, kind distribution.Kind, system, architecture string) (string, error) {
	if kind != distribution.TUI && kind != distribution.Full {
		return "", faults.Errorf(diagcatalog.UpgradeUnsupportedDistribution)
	}
	parts := releaseTag.FindStringSubmatch(version)
	if len(parts) != 4 {
		return "", faults.Errorf(diagcatalog.UpgradeUnsupportedReleaseVersion)
	}
	if _, err := time.Parse("20060102", parts[3]); err != nil {
		return "", faults.Errorf(diagcatalog.UpgradeInvalidReleaseDate)
	}
	if system == "darwin" {
		system = "macos"
	}
	if system != "macos" && system != "linux" || architecture != "amd64" && architecture != "arm64" {
		return "", faults.Errorf(diagcatalog.UpgradeUnsupportedReleasePlatform)
	}
	return "meca_" + strings.TrimPrefix(version, "v") + "_" + string(kind) + "_" + system + "_" + architecture + ".tar.gz", nil
}

func archiveChecksum(manifest, name string) (string, error) {
	var result string
	for line := range strings.SplitSeq(manifest, "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 || fields[1] != name {
			continue
		}
		decoded, err := hex.DecodeString(fields[0])
		if result != "" || err != nil || len(decoded) != sha256.Size {
			return "", faults.Errorf(diagcatalog.UpgradeInvalidOrDuplicateArchiveChecksum)
		}
		result = hex.EncodeToString(decoded)
	}
	if result == "" {
		return "", faults.Errorf(diagcatalog.UpgradeArchiveChecksumIsMissing)
	}
	return result, nil
}
