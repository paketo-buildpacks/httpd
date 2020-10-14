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

func testLogging(t *testing.T, when spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		pack   occam.Pack
		docker occam.Docker

		name   string
		source string
		image  occam.Image
	)

	it.Before(func() {
		pack = occam.NewPack().WithVerbose().WithNoColor()
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

		buildpackVersion, err := GetGitVersion()
		Expect(err).ToNot(HaveOccurred())

		Expect(logs).To(ContainLines(
			fmt.Sprintf("%s %s", buildpackInfo.Buildpack.Name, buildpackVersion),
			"  Resolving Apache HTTP Server version",
			"    Candidate version sources (in priority order):",
			`      buildpack.yml -> "2.4.*"`,
			"",
			MatchRegexp(`    Selected Apache HTTP Server version \(using buildpack\.yml\): 2\.4\.\d+`),
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
}
