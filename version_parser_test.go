package httpd_test

import (
	"os"
	"testing"

	"github.com/paketo-buildpacks/httpd"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testVersionParser(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		versionParser httpd.VersionParser
	)

	it.Before(func() {
		versionParser = httpd.NewVersionParser()
	})

	context("ParseVersion", func() {
		context("when there is no buildpack.yml", func() {
			it("returns a * for the version and empty version source", func() {
				version, versionSource, err := versionParser.ParseVersion("some-path")
				Expect(err).NotTo(HaveOccurred())
				Expect(version).To(Equal("*"))
				Expect(versionSource).To(Equal(""))
			})
		})

		context("when there is a buildpack.yml", func() {
			var path string

			it.Before(func() {
				file, err := os.CreateTemp("", "buildpack.yml")
				Expect(err).NotTo(HaveOccurred())

				_, err = file.WriteString(`{"httpd": {"version": "some-version"}}`)
				Expect(err).NotTo(HaveOccurred())

				path = file.Name()

				Expect(file.Close()).To(Succeed())
			})

			it.After(func() {
				Expect(os.Remove(path)).To(Succeed())
			})

			it("parses the version", func() {
				version, versionSource, err := versionParser.ParseVersion(path)
				Expect(err).NotTo(HaveOccurred())
				Expect(version).To(Equal("some-version"))
				Expect(versionSource).To(Equal("buildpack.yml"))
			})

			context("when there is not httpd version in the buildpack.yml", func() {
				it.Before(func() {
					err := os.WriteFile(path, []byte(`{"some-thing": {"version": "some-version"}}`), 0644)
					Expect(err).NotTo(HaveOccurred())
				})

				it("returns a * for the version and empty version source", func() {
					version, versionSource, err := versionParser.ParseVersion(path)
					Expect(err).NotTo(HaveOccurred())
					Expect(version).To(Equal("*"))
					Expect(versionSource).To(Equal(""))
				})
			})

			context("failure cases", func() {
				context("when the file cannot be opened", func() {
					it.Before(func() {
						Expect(os.Chmod(path, 0000)).To(Succeed())
					})

					it("returns an error", func() {
						_, _, err := versionParser.ParseVersion(path)
						Expect(err).To(MatchError(ContainSubstring("failed to parse buildpack.yml")))
						Expect(err).To(MatchError(ContainSubstring("permission denied")))
					})
				})

				context("when the file contains malformed yaml", func() {
					it.Before(func() {
						err := os.WriteFile(path, []byte("%%%"), 0644)
						Expect(err).NotTo(HaveOccurred())
					})

					it("returns an error", func() {
						_, _, err := versionParser.ParseVersion(path)
						Expect(err).To(MatchError(ContainSubstring("failed to parse buildpack.yml")))
						Expect(err).To(MatchError(ContainSubstring("could not find expected directive name")))
					})
				})
			})
		})
	})
}
