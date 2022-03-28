package integration_test

import (
	"fmt"
	"io"
	"net/http"
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
	})

	it.After(func() {
		Expect(docker.Container.Remove.Execute(container.ID)).To(Succeed())
		Expect(docker.Image.Remove.Execute(image.ID)).To(Succeed())
		Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())
		Expect(os.RemoveAll(source)).To(Succeed())
	})

	context("standard app source", func() {
		it.Before(func() {
			var err error
			source, err = occam.Source(filepath.Join("testdata", "zero_config"))
			Expect(err).NotTo(HaveOccurred())
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

		context("when the user sets a push state", func() {
			it("serves a static site that always serves index.html no matter the route", func() {
				var err error
				image, _, err = pack.Build.
					WithPullPolicy("never").
					WithBuildpacks(httpdBuildpack).
					WithEnv(map[string]string{
						"BP_WEB_SERVER":                   "httpd",
						"BP_WEB_SERVER_ENABLE_PUSH_STATE": "true",
					}).
					Execute(name, source)
				Expect(err).NotTo(HaveOccurred())

				container, err = docker.Container.Run.
					WithEnv(map[string]string{"PORT": "8080"}).
					WithPublish("8080").
					WithPublishAll().
					Execute(image.ID)
				Expect(err).NotTo(HaveOccurred())

				Eventually(container).Should(Serve(ContainSubstring("Hello World!")).OnPort(8080).WithEndpoint("/test"))
			})
		})

		context("when the user sets https forced redirect", func() {
			it("serves a static site that always redirects to https", func() {
				var err error
				image, _, err = pack.Build.
					WithPullPolicy("never").
					WithBuildpacks(httpdBuildpack).
					WithEnv(map[string]string{
						"BP_WEB_SERVER":             "httpd",
						"BP_WEB_SERVER_FORCE_HTTPS": "true",
					}).
					Execute(name, source)
				Expect(err).NotTo(HaveOccurred())

				container, err = docker.Container.Run.
					WithEnv(map[string]string{"PORT": "8080"}).
					WithPublish("8080").
					WithPublishAll().
					Execute(image.ID)
				Expect(err).NotTo(HaveOccurred())

				client := &http.Client{
					CheckRedirect: func(req *http.Request, via []*http.Request) error {
						return http.ErrUseLastResponse
					},
				}

				response, err := client.Get(fmt.Sprintf("http://localhost:%s", container.HostPort("8080")))
				Expect(err).NotTo(HaveOccurred())
				Expect(response.StatusCode).To(Equal(http.StatusMovedPermanently))

				contents, err := io.ReadAll(response.Body)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(contents)).To(ContainSubstring(fmt.Sprintf("https://localhost:%s", container.HostPort("8080"))))
			})
		})
	})

	context("app with binding", func() {
		it.Before(func() {
			var err error
			source, err = occam.Source(filepath.Join("testdata", "zero_config_basic_auth"))
			Expect(err).NotTo(HaveOccurred())
		})

		it("serves up a static site that requires basic auth", func() {
			var err error
			image, _, err = pack.Build.
				WithPullPolicy("never").
				WithBuildpacks(httpdBuildpack).
				WithEnv(map[string]string{
					"BP_WEB_SERVER":        "httpd",
					"SERVICE_BINDING_ROOT": "/bindings",
				}).
				WithVolumes(fmt.Sprintf("%s:/bindings/auth", filepath.Join(source, "binding"))).
				Execute(name, filepath.Join(source, "app"))
			Expect(err).NotTo(HaveOccurred())

			container, err = docker.Container.Run.
				WithEnv(map[string]string{
					"PORT":                 "8080",
					"SERVICE_BINDING_ROOT": "/bindings",
				}).
				WithVolumes(fmt.Sprintf("%s:/bindings/auth", filepath.Join(source, "binding"))).
				WithPublish("8080").
				WithPublishAll().
				Execute(image.ID)
			Expect(err).NotTo(HaveOccurred())

			response, err := http.Get(fmt.Sprintf("http://localhost:%s", container.HostPort("8080")))
			Expect(err).NotTo(HaveOccurred())
			Expect(response.StatusCode).To(Equal(http.StatusUnauthorized))

			req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://localhost:%s", container.HostPort("8080")), http.NoBody)
			Expect(err).NotTo(HaveOccurred())

			req.SetBasicAuth("user", "password")

			response, err = http.DefaultClient.Do(req)
			Expect(err).NotTo(HaveOccurred())

			contents, err := io.ReadAll(response.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(contents)).To(ContainSubstring("Hello World!"))
		})
	})
}
