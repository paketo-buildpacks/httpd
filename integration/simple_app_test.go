package integration_test

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/paketo-buildpacks/occam"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
	. "github.com/paketo-buildpacks/occam/matchers"
)

func testSimpleApp(t *testing.T, context spec.G, it spec.S) {
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
	})

	it.After(func() {
		Expect(docker.Container.Remove.Execute(container.ID)).To(Succeed())
		Expect(docker.Image.Remove.Execute(image.ID)).To(Succeed())
		Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())
		Expect(os.RemoveAll(source)).To(Succeed())
	})

	it("serves up staticfile", func() {
		var err error

		source, err = occam.Source(filepath.Join("testdata", "simple_app"))
		Expect(err).NotTo(HaveOccurred())

		image, _, err = pack.Build.
			WithBuildpacks(httpdBuildpack).
			WithPullPolicy("never").
			Execute(name, source)
		Expect(err).NotTo(HaveOccurred())

		container, err = docker.Container.Run.
			WithEnv(map[string]string{"PORT": "8080"}).
			WithPublish("8080").
			WithPublishAll().
			Execute(image.ID)
		Expect(err).NotTo(HaveOccurred())

		Eventually(container).Should(BeAvailable())

		response, err := http.Get(fmt.Sprintf("http://localhost:%s", container.HostPort("8080")))
		Expect(err).NotTo(HaveOccurred())
		Expect(response.StatusCode).To(Equal(http.StatusOK))
	})

	context("when BP_LIVE_RELOAD_ENABLED=true", func() {
		var noReloadContainer occam.Container

		it.Before(func() {
			var err error
			source, err = occam.Source(filepath.Join("testdata", "simple_app"))
			Expect(err).NotTo(HaveOccurred())
		})

		it.After(func() {
			Expect(docker.Container.Remove.Execute(noReloadContainer.ID)).To(Succeed())
		})

		it("adds a reloadable process type as the default process", func() {
			var err error
			var logs fmt.Stringer

			image, logs, err = pack.Build.
				WithBuildpacks(
					watchexecBuildpack,
					httpdBuildpack,
				).
				WithPullPolicy("never").
				WithEnv(map[string]string{
					"BP_LIVE_RELOAD_ENABLED": "true",
				}).
				Execute(name, source)
			Expect(err).NotTo(HaveOccurred())

			container, err = docker.Container.Run.
				WithEnv(map[string]string{"PORT": "8080"}).
				WithPublish("8080").
				WithPublishAll().
				Execute(image.ID)
			Expect(err).NotTo(HaveOccurred())

			Eventually(container).Should(BeAvailable())

			response, err := http.Get(fmt.Sprintf("http://localhost:%s", container.HostPort("8080")))
			Expect(err).NotTo(HaveOccurred())
			Expect(response.StatusCode).To(Equal(http.StatusOK))

			noReloadContainer, err = docker.Container.Run.
				WithEnv(map[string]string{"PORT": "8080"}).
				WithPublish("8080").
				WithPublishAll().
				WithEntrypoint("no-reload").
				Execute(image.ID)
			Expect(err).NotTo(HaveOccurred())

			Expect(logs).To(ContainLines("  Assigning launch processes:"))
			Expect(logs).To(ContainLines("    web (default): watchexec --restart --watch /workspace --shell none -- httpd -f /workspace/httpd.conf -k start -DFOREGROUND"))
			Expect(logs).To(ContainLines("    no-reload:     httpd -f /workspace/httpd.conf -k start -DFOREGROUND"))
		})
	})
}
