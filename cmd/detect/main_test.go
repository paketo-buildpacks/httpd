package main

import (
	"github.com/buildpack/libbuildpack/buildplan"
	"github.com/cloudfoundry/httpd-cnb/httpd"
	"gopkg.in/yaml.v2"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/libcfbuildpack/detect"
	"github.com/cloudfoundry/libcfbuildpack/test"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitDetect(t *testing.T) {
	RegisterTestingT(t)
	spec.Run(t, "Detect", testDetect, spec.Report(report.Terminal{}))
}

func testDetect(t *testing.T, when spec.G, it spec.S) {
	var factory *test.DetectFactory

	it.Before(func() {
		factory = test.NewDetectFactory(t)
	})

	when("there is an httpd.conf", func() {
		it.Before(func() {
			test.WriteFile(t, filepath.Join(factory.Detect.Application.Root, "httpd.conf"),"")
		})

		it("should pass with the default version of httpd", func() {
			code, err := runDetect(factory.Detect)
			Expect(err).NotTo(HaveOccurred())

			Expect(code).To(Equal(detect.PassStatusCode))

			Expect(factory.Output).To(Equal(buildplan.BuildPlan{
				httpd.Dependency: buildplan.Dependency{
					Metadata: buildplan.Metadata{"launch": true},
				},
			}))
		})

		when("there is a buildpack.yml", func() {
			it("should request the supplied version", func() {
				buildpackYAML := BuildpackYAML{
					Config: httpd.Config{
						Version: "1.2.3",
					},
				}
				buf, _ := yaml.Marshal(buildpackYAML)

				test.WriteFile(t, filepath.Join(factory.Detect.Application.Root, "buildpack.yml"), string(buf))

				code, err := runDetect(factory.Detect)
				Expect(err).NotTo(HaveOccurred())

				Expect(code).To(Equal(detect.PassStatusCode))

				Expect(factory.Output).To(Equal(buildplan.BuildPlan{
					httpd.Dependency: buildplan.Dependency{
						Version:  "1.2.3",
						Metadata: buildplan.Metadata{"launch": true},
					},
				}))
			})

			it("should request the default version when no version is requested", func() {
				test.WriteFile(t, filepath.Join(factory.Detect.Application.Root, "buildpack.yml"), " ")

				code, err := runDetect(factory.Detect)
				Expect(err).NotTo(HaveOccurred())

				Expect(code).To(Equal(detect.PassStatusCode))

				Expect(factory.Output).To(Equal(buildplan.BuildPlan{
					httpd.Dependency: buildplan.Dependency{
						Metadata: buildplan.Metadata{"launch": true},
					},
				}))
			})
		})
	})

	when("there is NOT an httpd.conf", func() {
		it("should not pass", func() {
			code, err := runDetect(factory.Detect)
			Expect(err).To(HaveOccurred())

			Expect(code).To(Equal(detect.FailStatusCode))
		})
	})
}
