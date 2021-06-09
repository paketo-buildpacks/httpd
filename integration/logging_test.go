package integration_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/paketo-buildpacks/occam"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
	. "github.com/paketo-buildpacks/occam/matchers"
)

func testLogging(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		pack   occam.Pack
		docker occam.Docker

		name   string
		source string
		image  occam.Image
	)

	it.Before(func() {
		pack = occam.NewPack().WithNoColor()
		docker = occam.NewDocker()

		var err error
		name, err = occam.RandomName()
		Expect(err).NotTo(HaveOccurred())
	})

	it.After(func() {
		Expect(docker.Image.Remove.Execute(image.ID)).To(Succeed())
		Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())
		Expect(os.RemoveAll(source)).To(Succeed())
	})

	it("logs the build process", func() {
		var (
			err  error
			logs fmt.Stringer
		)

		source, err = occam.Source(filepath.Join("testdata", "buildpack_yaml"))
		Expect(err).NotTo(HaveOccurred())

		image, logs, err = pack.Build.
			WithBuildpacks(httpdBuildpack).
			WithPullPolicy("never").
			Execute(name, source)
		Expect(err).NotTo(HaveOccurred())

		Expect(logs).To(ContainLines(
			MatchRegexp(fmt.Sprintf(`%s \d+\.\d+\.\d+`, buildpackInfo.Buildpack.Name)),
			"  Resolving Apache HTTP Server version",
			"    Candidate version sources (in priority order):",
			`      buildpack.yml -> "2.4.*"`,
			"",
			MatchRegexp(`    Selected Apache HTTP Server version \(using buildpack\.yml\): 2\.4\.\d+`),
			"",
			"    WARNING: Setting the server version through buildpack.yml will be deprecated soon in Apache HTTP Server Buildpack v2.0.0.",
			"    Please specify the version through the $BP_HTTPD_VERSION environment variable instead. See docs for more information.",
			"",
			"  Executing build process",
			MatchRegexp(`    Installing Apache HTTP Server \d+\.\d+\.\d+`),
			MatchRegexp(`      Completed in (\d+\.\d+|\d{3})`),
			"",
			"  Configuring environment",
			`    APP_ROOT    -> "/workspace"`,
			fmt.Sprintf(`    SERVER_ROOT -> "/layers/%s/httpd"`, strings.ReplaceAll(buildpackInfo.Buildpack.ID, "/", "_")),
		))
	})

	context("the app is built with BP_HTTPD_VERSION set", func() {
		it("builds the app with the specified version", func() {
			var (
				err  error
				logs fmt.Stringer
			)

			source, err = occam.Source(filepath.Join("testdata", "buildpack_yaml"))
			Expect(err).NotTo(HaveOccurred())

			lowestVersion := buildpackInfo.Metadata.Dependencies[0].Version
			image, logs, err = pack.Build.
				WithBuildpacks(httpdBuildpack).
				WithPullPolicy("never").
				WithEnv(map[string]string{
					"BP_HTTPD_VERSION": lowestVersion,
				}).
				Execute(name, source)
			Expect(err).NotTo(HaveOccurred())

			Expect(logs).To(ContainLines(
				MatchRegexp(fmt.Sprintf(`%s \d+\.\d+\.\d+`, buildpackInfo.Buildpack.Name)),
				"  Resolving Apache HTTP Server version",
				"    Candidate version sources (in priority order):",
				fmt.Sprintf(`      BP_HTTPD_VERSION -> "%s"`, lowestVersion),
				`      buildpack.yml    -> "2.4.*"`,
				"",
				MatchRegexp(fmt.Sprintf(`    Selected Apache HTTP Server version \(using BP_HTTPD_VERSION\): %s`, lowestVersion)),
				"",
				"  Executing build process",
				MatchRegexp(`    Installing Apache HTTP Server \d+\.\d+\.\d+`),
				MatchRegexp(`      Completed in (\d+\.\d+|\d{3})`),
				"",
				"  Configuring environment",
				`    APP_ROOT    -> "/workspace"`,
				fmt.Sprintf(`    SERVER_ROOT -> "/layers/%s/httpd"`, strings.ReplaceAll(buildpackInfo.Buildpack.ID, "/", "_")),
			))

		})

	})
}
