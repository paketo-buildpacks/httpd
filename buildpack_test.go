package httpd_test

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/paketo-buildpacks/httpd"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testBuildpack(t *testing.T, context spec.G, it spec.S) {
	var Expect = NewWithT(t).Expect

	context("ParseBuildpack", func() {
		var path string

		it.Before(func() {
			file, err := ioutil.TempFile("", "buildpack.yml")
			Expect(err).NotTo(HaveOccurred())

			_, err = file.WriteString(`{"httpd": {"version": "some-version"}}`)
			Expect(err).NotTo(HaveOccurred())

			path = file.Name()

			Expect(file.Close()).To(Succeed())
		})

		it.After(func() {
			Expect(os.Remove(path)).To(Succeed())
		})

		it("parses a buildpack config", func() {
			buildpack, err := httpd.ParseBuildpack(path)
			Expect(err).NotTo(HaveOccurred())
			Expect(buildpack).To(Equal(httpd.Buildpack{
				HTTPD: httpd.BuildpackHTTPD{
					Version: "some-version",
				},
			}))
		})

		context("failure cases", func() {
			context("when the file does not exist", func() {
				it("returns an error", func() {
					_, err := httpd.ParseBuildpack("missing-file")
					Expect(err).To(MatchError(ContainSubstring("failed to parse buildpack.yml")))
					Expect(err).To(MatchError(ContainSubstring("no such file or directory")))
				})
			})

			context("when the file contains malformed yaml", func() {
				it.Before(func() {
					err := ioutil.WriteFile(path, []byte("%%%"), 0644)
					Expect(err).NotTo(HaveOccurred())
				})

				it("returns an error", func() {
					_, err := httpd.ParseBuildpack(path)
					Expect(err).To(MatchError(ContainSubstring("failed to parse buildpack.yml")))
					Expect(err).To(MatchError(ContainSubstring("could not find expected directive name")))
				})
			})
		})
	})
}
