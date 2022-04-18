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

func testCaching(t *testing.T, when spec.G, it spec.S) {
	var (
		Expect     = NewWithT(t).Expect
		Eventually = NewWithT(t).Eventually

		pack   occam.Pack
		docker occam.Docker

		name         string
		source       string
		imageIDs     map[string]struct{}
		containerIDs map[string]struct{}
	)

	it.Before(func() {
		pack = occam.NewPack().WithNoColor()
		docker = occam.NewDocker()

		var err error
		name, err = occam.RandomName()
		Expect(err).NotTo(HaveOccurred())

		imageIDs = map[string]struct{}{}
		containerIDs = map[string]struct{}{}
	})

	it.After(func() {
		for id := range containerIDs {
			Expect(docker.Container.Remove.Execute(id)).To(Succeed())
		}

		for id := range imageIDs {
			Expect(docker.Image.Remove.Execute(id)).To(Succeed())
		}

		Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())
		Expect(os.RemoveAll(source)).To(Succeed())
	})

	it("uses a cached layer and doesn't run twice", func() {
		source, err := occam.Source(filepath.Join("testdata", "simple_app"))
		Expect(err).ToNot(HaveOccurred())

		build := pack.Build.WithBuildpacks(httpdBuildpack)

		firstImage, logs, err := build.Execute(name, source)
		Expect(err).NotTo(HaveOccurred())

		Expect(logs).To(ContainLines(
			MatchRegexp(fmt.Sprintf(`%s \d+\.\d+\.\d+`, buildpackInfo.Buildpack.Name)),
			"  Resolving Apache HTTP Server version",
			"    Candidate version sources (in priority order):",
			`      <unknown> -> "*"`,
			"",
			MatchRegexp(`    Selected Apache HTTP Server version \(using \<unknown\>\): 2\.4\.\d+`),
			"",
			"  Executing build process",
			MatchRegexp(`    Installing Apache HTTP Server \d+\.\d+\.\d+`),
			MatchRegexp(`      Completed in (\d+\.\d+|\d{3})`),
			"",
			"  Configuring launch environment",
			`    APP_ROOT    -> "/workspace"`,
			fmt.Sprintf(`    SERVER_ROOT -> "/layers/%s/httpd"`, strings.ReplaceAll(buildpackInfo.Buildpack.ID, "/", "_")),
			"",
			"  Assigning launch processes:",
			"    web (default): httpd -f /workspace/httpd.conf -k start -DFOREGROUND",
		))

		imageIDs[firstImage.ID] = struct{}{}

		Expect(firstImage.Buildpacks).To(HaveLen(1))
		Expect(firstImage.Buildpacks[0].Key).To(Equal(buildpackInfo.Buildpack.ID))
		Expect(firstImage.Buildpacks[0].Layers).To(HaveKey("httpd"))

		container, err := docker.Container.Run.
			WithEnv(map[string]string{"PORT": "8080"}).
			WithPublish("8080").
			WithPublishAll().
			Execute(firstImage.ID)
		Expect(err).NotTo(HaveOccurred())

		containerIDs[container.ID] = struct{}{}

		Eventually(container).Should(BeAvailable())

		secondImage, logs, err := build.Execute(name, source)
		Expect(err).NotTo(HaveOccurred())

		Expect(logs).To(ContainLines(
			MatchRegexp(fmt.Sprintf(`%s \d+\.\d+\.\d+`, buildpackInfo.Buildpack.Name)),
			"  Resolving Apache HTTP Server version",
			"    Candidate version sources (in priority order):",
			`      <unknown> -> "*"`,
			"",
			MatchRegexp(`    Selected Apache HTTP Server version \(using \<unknown\>\): 2\.4\.\d+`),
			"",
			fmt.Sprintf("  Reusing cached layer /layers/%s/httpd", strings.ReplaceAll(buildpackInfo.Buildpack.ID, "/", "_")),
			"",
			"  Assigning launch processes:",
			"    web (default): httpd -f /workspace/httpd.conf -k start -DFOREGROUND",
		))

		imageIDs[secondImage.ID] = struct{}{}

		Expect(secondImage.Buildpacks).To(HaveLen(1))
		Expect(secondImage.Buildpacks[0].Key).To(Equal(buildpackInfo.Buildpack.ID))
		Expect(secondImage.Buildpacks[0].Layers).To(HaveKey("httpd"))

		container, err = docker.Container.Run.
			WithEnv(map[string]string{"PORT": "8080"}).
			WithPublish("8080").
			WithPublishAll().
			Execute(secondImage.ID)
		Expect(err).NotTo(HaveOccurred())

		containerIDs[container.ID] = struct{}{}

		Eventually(container).Should(BeAvailable())

		Expect(secondImage.Buildpacks[0].Layers["httpd"].SHA).To(Equal(firstImage.Buildpacks[0].Layers["httpd"].SHA))
		Expect(secondImage.ID).To(Equal(firstImage.ID))
	})
}
