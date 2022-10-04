package main

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/joshuatcasey/libdependency/retrieve"
	"github.com/joshuatcasey/libdependency/versionology"
	"github.com/paketo-buildpacks/packit/v2/cargo"
	"github.com/paketo-buildpacks/packit/v2/vacation"
)

type HttpdMetadata struct {
	SemverVersion *semver.Version
}

type HttpdRelease struct {
	version       string
	releaseDate   time.Time
	dependencyURL string
	sha256URL     string
	sha1URL       string
	md5URL        string
}

func (httpdMetadata HttpdMetadata) Version() *semver.Version {
	return httpdMetadata.SemverVersion
}

func main() {
	retrieve.NewMetadata("httpd", getHttpdVersions, generateMetadata)
}

func getReleases(versionFilter string) ([]HttpdRelease, error) {
	filePattern := "httpd-*.tar.bz2*"
	if versionFilter != "" {
		filePattern = fmt.Sprintf("httpd-%s.tar.bz2*", versionFilter)
	}

	resp, err := http.Get("http://archive.apache.org/dist/httpd/?F=2&C=M&O=D&P=" + filePattern)
	if err != nil {
		return nil, fmt.Errorf("could not get file list from archive.apache.org: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)

	re := regexp.MustCompile(`>httpd-([\d\.]+)\.tar\.bz2<.*(\d\d\d\d-\d\d-\d\d \d\d:\d\d)`)

	var releases []HttpdRelease
	for _, line := range strings.Split(string(body), "\n") {
		matches := re.FindStringSubmatch(line)
		if len(matches) != 3 {
			continue
		}

		version := matches[1]

		date, err := time.Parse("2006-01-02 15:04", matches[2])
		if err != nil {
			return nil, fmt.Errorf("could not parse '%s' as date for version '%s'", matches[2], version)
		}

		releases = append(releases, HttpdRelease{
			version:       version,
			releaseDate:   date,
			dependencyURL: fmt.Sprintf("http://archive.apache.org/dist/httpd/httpd-%s.tar.bz2", version),
			sha256URL:     checksumURL(string(body), version, "sha256"),
			sha1URL:       checksumURL(string(body), version, "sha1"),
			md5URL:        checksumURL(string(body), version, "md5"),
		})
	}

	return releases, nil
}

func checksumURL(index string, version string, checksum string) string {
	checksumFilename := fmt.Sprintf("httpd-%s.tar.bz2.%s", version, checksum)
	if strings.Contains(index, checksumFilename) {
		return fmt.Sprintf("http://archive.apache.org/dist/httpd/%s", checksumFilename)
	}
	return ""
}

func sortReleases(releases []HttpdRelease) error {
	var sortErr error

	sort.Slice(releases, func(i, j int) bool {
		if releases[i].releaseDate != releases[j].releaseDate {
			return releases[i].releaseDate.After(releases[j].releaseDate)
		}

		var v1, v2 *semver.Version

		v1, sortErr = semver.NewVersion(releases[i].version)
		if sortErr != nil {
			return false
		}

		v2, sortErr = semver.NewVersion(releases[j].version)
		if sortErr != nil {
			return false
		}

		return v1.GreaterThan(v2)
	})

	return sortErr
}

func getHttpdVersions() (versionology.VersionFetcherArray, error) {
	releases, err := getReleases("")
	if err != nil {
		return nil, fmt.Errorf("could not get releases: %w", err)
	}

	err = sortReleases(releases)
	if err != nil {
		return nil, fmt.Errorf("could not sort releases: %w", err)
	}

	var versions []versionology.VersionFetcher
	for _, release := range releases {
		versions = append(versions, HttpdMetadata{
			semver.MustParse(release.version),
		})
	}

	return versions, nil
}

func generateMetadata(hasVersion versionology.VersionFetcher) ([]versionology.Dependency, error) {
	httpdVersion := hasVersion.Version().String()

	releases, err := getReleases(httpdVersion)
	release := releases[0]

	sourceSHA, err := getDependencySHA(release)
	if err != nil {
		return nil, fmt.Errorf("could get sha: %w", err)
	}

	// If the dependency is to be compiled, the SHA256 and URI field from the metadata should be omitted in this step.
	dep := cargo.ConfigMetadataDependency{
		Version:         httpdVersion,
		ID:              "httpd",
		Name:            "HTTPD",
		Source:          release.dependencyURL,
		SourceChecksum:  sourceSHA,
		DeprecationDate: nil,
		Licenses:        retrieve.LookupLicenses(release.dependencyURL, decompress),
		PURL:            retrieve.GeneratePURL("httpd", httpdVersion, sourceSHA, release.dependencyURL),
		CPE:             fmt.Sprintf("cpe:2.3:a:apache:http_server:%s:*:*:*:*:*:*:*", httpdVersion),
		Stacks:          []string{"io.buildpacks.stacks.bionic"},
	}

	bionicDependency, err := versionology.NewDependency(dep, "bionic")
	if err != nil {
		return nil, fmt.Errorf("could get sha: %w", err)
	}

	dep.Stacks = []string{"io.buildpacks.stacks.jammy"}

	jammyDependency, err := versionology.NewDependency(dep, "jammy")
	if err != nil {
		return nil, fmt.Errorf("could get sha: %w", err)
	}

	return []versionology.Dependency{bionicDependency, jammyDependency}, nil

}

func dependencyVersionIsMissingChecksum(version string) bool {
	versionsWithMissingChecksum := map[string]bool{
		"2.2.3": true,
	}

	_, shouldBeIgnored := versionsWithMissingChecksum[version]
	return shouldBeIgnored
}

func getDependencySHA(release HttpdRelease) (string, error) {
	if release.sha256URL == "" && release.sha1URL == "" && release.md5URL == "" && !dependencyVersionIsMissingChecksum(release.version) {
		return "", errors.New("could not find checksum file")
	}

	if release.sha256URL != "" {
		resp, err := http.Get(release.sha256URL)
		if err != nil {
			return "", fmt.Errorf("could not make request: %w", err)
		}
		defer resp.Body.Close()
		checksumContents, err := io.ReadAll(resp.Body)

		return strings.Fields(string(checksumContents))[0], nil
	}

	tempDir, err := os.MkdirTemp("", "httpd")
	if err != nil {
		return "", fmt.Errorf("could not make temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	dependencyPath := filepath.Join(tempDir, filepath.Base(release.dependencyURL))

	resp, err := http.Get(release.dependencyURL)
	if err != nil {
		return "", fmt.Errorf("could not make request: %w", err)
	}
	defer resp.Body.Close()

	file, err := os.OpenFile(dependencyPath, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		file.Close()
		return "", fmt.Errorf("failed to write to file: %w", err)
	}

	err = file.Close()
	if err != nil {
		return "", fmt.Errorf("failed to close file: %w", err)
	}

	err = verifyChecksum(release, dependencyPath)
	if err != nil {
		return "", fmt.Errorf("could not verify checksum: %w", err)
	}

	sha256, err := getSHA256(dependencyPath)
	if err != nil {
		return "", fmt.Errorf("could not get sha256: %w", err)
	}

	return sha256, nil
}

func verifyChecksum(release HttpdRelease, dependencyPath string) error {
	if dependencyVersionIsMissingChecksum(release.version) {
		return nil
	}

	if release.sha1URL != "" {
		resp, err := http.Get(release.sha256URL)
		if err != nil {
			return fmt.Errorf("could not make request: %w", err)
		}
		defer resp.Body.Close()

		checksumContents, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("could not download sha1 file: %w", err)
		}

		fields := strings.Fields(string(checksumContents))

		var checksum string
		if strings.HasPrefix(fields[0], "SHA1") {
			checksum = fields[len(fields)-1]
		} else {
			checksum = fields[0]
		}

		err = verifySHA1(dependencyPath, checksum)
		if err != nil {
			return fmt.Errorf("could not verify sha1: %w", err)
		}
	} else if release.md5URL != "" {
		resp, err := http.Get(release.md5URL)
		if err != nil {
			return fmt.Errorf("could not make request: %w", err)
		}
		defer resp.Body.Close()

		checksumContents, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("could not download md5 file: %w", err)
		}

		fields := strings.Fields(string(checksumContents))

		var checksum string
		if strings.HasPrefix(fields[0], "MD5") {
			checksum = fields[len(fields)-1]
		} else {
			checksum = fields[0]
		}

		err = verifyMD5(dependencyPath, checksum)
		if err != nil {
			return fmt.Errorf("could not verify md5: %w", err)
		}
	}

	return nil
}

func getSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "nil", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	hash := sha256.New()
	_, err = io.Copy(hash, file)
	if err != nil {
		return "nil", fmt.Errorf("failed to calculate SHA256: %w", err)
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func getMD5(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "nil", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	hash := md5.New()
	_, err = io.Copy(hash, file)
	if err != nil {
		return "nil", fmt.Errorf("failed to calculate MD5: %w", err)
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func verifyMD5(path, expectedMD5 string) error {
	actualMD5, err := getMD5(path)
	if err != nil {
		return fmt.Errorf("failed to get actual MD5: %w", err)
	}

	if actualMD5 != expectedMD5 {
		return fmt.Errorf("expected MD5 '%s' but got '%s'", expectedMD5, actualMD5)
	}

	return nil
}

func getSHA1(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "nil", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	hash := sha1.New()
	_, err = io.Copy(hash, file)
	if err != nil {
		return "nil", fmt.Errorf("failed to calculate SHA1: %w", err)
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func verifySHA1(path, expectedSHA string) error {
	actualSHA, err := getSHA1(path)
	if err != nil {
		return fmt.Errorf("failed to get actual SHA256: %w", err)
	}

	if actualSHA != expectedSHA {
		return fmt.Errorf("expected SHA256 '%s' but got '%s'", expectedSHA, actualSHA)
	}

	return nil
}

func decompress(artifact io.Reader, destination string) error {
	archive := vacation.NewArchive(artifact)

	err := archive.StripComponents(1).Decompress(destination)
	if err != nil {
		return fmt.Errorf("failed to decompress source file: %w", err)
	}

	return nil
}
