package integration_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/paketo-buildpacks/occam"
	"github.com/paketo-buildpacks/packit/v2/fs"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
	. "github.com/paketo-buildpacks/occam/matchers"
)

func testZeroConfig(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect     = NewWithT(t).Expect
		Eventually = NewWithT(t).Eventually

		pack   occam.Pack
		docker occam.Docker

		name      string
		source    string
		image     occam.Image
		container occam.Container
	)

	it.Before(func() {
		pack = occam.NewPack().WithVerbose()
		docker = occam.NewDocker()

		var err error
		name, err = occam.RandomName()
		Expect(err).NotTo(HaveOccurred())

		source, err = occam.Source(filepath.Join("testdata", "zero_config"))
		Expect(err).NotTo(HaveOccurred())
	})

	it.After(func() {
		Expect(docker.Container.Remove.Execute(container.ID)).To(Succeed())
		Expect(docker.Image.Remove.Execute(image.ID)).To(Succeed())
		Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())
		Expect(os.RemoveAll(source)).To(Succeed())
	})

	it("serves up uses default config", func() {
		var err error
		image, _, err = pack.Build.
			WithPullPolicy("never").
			WithBuildpacks(httpdBuildpack).
			WithEnv(map[string]string{
				"BP_WEB_SERVER": "httpd",
			}).
			Execute(name, source)
		Expect(err).NotTo(HaveOccurred())

		container, err = docker.Container.Run.
			WithEnv(map[string]string{"PORT": "8080"}).
			WithPublish("8080").
			WithPublishAll().
			Execute(image.ID)
		Expect(err).NotTo(HaveOccurred())

		Eventually(container).Should(Serve(ContainSubstring("Hello World!")).OnPort(8080))
	})

	context("when the static directory is configured to something other than public", func() {
		it.Before(func() {
			Expect(fs.Move(filepath.Join(source, "public"), filepath.Join(source, "htdocs"))).To(Succeed())
		})

		it("serves a static site", func() {
			var err error
			image, _, err = pack.Build.
				WithPullPolicy("never").
				WithBuildpacks(httpdBuildpack).
				WithEnv(map[string]string{
					"BP_WEB_SERVER":      "httpd",
					"BP_WEB_SERVER_ROOT": "htdocs",
				}).
				Execute(name, source)
			Expect(err).NotTo(HaveOccurred())

			container, err = docker.Container.Run.
				WithEnv(map[string]string{"PORT": "8080"}).
				WithPublish("8080").
				WithPublishAll().
				Execute(image.ID)
			Expect(err).NotTo(HaveOccurred())

			Eventually(container).Should(Serve(ContainSubstring("Hello World!")).OnPort(8080))
		})
	})
}