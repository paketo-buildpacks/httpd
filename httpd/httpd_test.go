package httpd

import (
	"github.com/cloudfoundry/libcfbuildpack/layers"
	"path/filepath"
	"testing"

	"github.com/buildpack/libbuildpack/buildplan"
	"github.com/sclevine/spec/report"

	"github.com/cloudfoundry/libcfbuildpack/test"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
)

func TestUnitHTTPD(t *testing.T) {
	RegisterTestingT(t)
	spec.Run(t, "HTTPD", testHTTPD, spec.Report(report.Terminal{}))
}

func testHTTPD(t *testing.T, when spec.G, it spec.S) {
	when("NewContributor", func() {
		var stubHTTPDFixture = filepath.Join("stub-httpd.tar.gz")

		it("returns true if a build plan exists", func() {
			f := test.NewBuildFactory(t)
			f.AddBuildPlan(t, Dependency, buildplan.Dependency{})
			f.AddDependency(t, Dependency, stubHTTPDFixture)

			_, willContribute, err := NewContributor(f.Build)
			Expect(err).NotTo(HaveOccurred())
			Expect(willContribute).To(BeTrue())
		})

		it("returns false if a build plan does not exist", func() {
			f := test.NewBuildFactory(t)

			_, willContribute, err := NewContributor(f.Build)
			Expect(err).NotTo(HaveOccurred())
			Expect(willContribute).To(BeFalse())
		})

		it("should contribute httpd to launch when launch is true", func() {
			f := test.NewBuildFactory(t)
			f.AddBuildPlan(t, Dependency, buildplan.Dependency{
				Metadata: buildplan.Metadata{"launch": true},
			})
			f.AddDependency(t, Dependency, stubHTTPDFixture)

			nodeContributor, _, err := NewContributor(f.Build)
			Expect(err).NotTo(HaveOccurred())

			err = nodeContributor.Contribute()
			Expect(err).NotTo(HaveOccurred())

			layer := f.Build.Layers.Layer(Dependency)
			test.BeLayerLike(t, layer, false, false, true)
			test.BeFileLike(t, filepath.Join(layer.Root, "stub.txt"), 0644, "This is a stub file\n")
			test.BeLaunchMetadataLike(t, f.Build.Layers, layers.Metadata{
				Processes: []layers.Process{
					{"web", `apachectl -f "httpd.conf" -k start -DFOREGROUND`},
				},
			})
		})
	})
}
