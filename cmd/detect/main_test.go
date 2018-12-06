package main

import (
	"github.com/buildpack/libbuildpack/buildplan"
	"github.com/cloudfoundry/httpd-cnb/httpd"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/libcfbuildpack/detect"
	"github.com/cloudfoundry/libcfbuildpack/test"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestDetect(t *testing.T) {
	RegisterTestingT(t)
	spec.Run(t, "Detect", testDetect, spec.Report(report.Terminal{}))
}

func testDetect(t *testing.T, when spec.G, it spec.S) {
	var (
		err     error
		dir     string
		factory *test.DetectFactory
	)

	it.Before(func() {
		dir, err = ioutil.TempDir("", "")
		Expect(err).NotTo(HaveOccurred())

		factory = test.NewDetectFactory(t)
		factory.Detect.Application.Root = dir
	})

	it.After(func() {
		Expect(os.RemoveAll(dir)).To(Succeed())
	})

	when("there is an httpd.conf", func() {
		it.Before(func() {
			Expect(ioutil.WriteFile(filepath.Join(dir, "httpd.conf"), []byte(""), 0666)).To(Succeed())
		})

		it("should pass with the default version of httpd", func() {
			code, err := runDetect(factory.Detect)
			Expect(err).NotTo(HaveOccurred())

			Expect(code).To(Equal(detect.PassStatusCode))

			test.BeBuildPlanLike(t, factory.Output, buildplan.BuildPlan{
				httpd.Dependency: buildplan.Dependency{
					Metadata: buildplan.Metadata{"launch": true},
				},
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
