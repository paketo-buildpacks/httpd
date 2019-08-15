/*
 * Copyright 2018-2019 the original author or authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package httpd

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/libcfbuildpack/buildpackplan"

	"github.com/cloudfoundry/libcfbuildpack/layers"

	"github.com/sclevine/spec/report"

	"github.com/cloudfoundry/libcfbuildpack/test"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
)

func TestUnitHTTPDContributor(t *testing.T) {
	spec.Run(t, "HTTPD Contributor", testHTTPDContributor, spec.Report(report.Terminal{}))
}

func testHTTPDContributor(t *testing.T, when spec.G, it spec.S) {
	it.Before(func() {
		RegisterTestingT(t)
	})

	when("NewContributor", func() {
		var stubHTTPDFixture = filepath.Join("fixtures", "stub-httpd.tar.gz")

		it("returns true if a build plan exists", func() {
			f := test.NewBuildFactory(t)
			f.AddPlan(buildpackplan.Plan{Name: Dependency})
			f.AddDependency(Dependency, stubHTTPDFixture)

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
			f.AddPlan(buildpackplan.Plan{
				Name:     Dependency,
				Metadata: buildpackplan.Metadata{"launch": true},
			})
			f.AddDependency(Dependency, stubHTTPDFixture)

			nodeContributor, _, err := NewContributor(f.Build)
			Expect(err).NotTo(HaveOccurred())

			Expect(nodeContributor.Contribute()).To(Succeed())

			layer := f.Build.Layers.Layer(Dependency)

			Expect(layer).To(test.HaveLayerMetadata(false, false, true))
			Expect(layer).To(test.HaveOverrideLaunchEnvironment("APP_ROOT", f.Build.Application.Root))
			Expect(layer).To(test.HaveOverrideLaunchEnvironment("SERVER_ROOT", layer.Root))
			Expect(filepath.Join(layer.Root, "stub.txt")).To(BeARegularFile())
			Expect(f.Build.Layers).To(test.HaveApplicationMetadata(
				layers.Metadata{Processes: []layers.Process{{"web", fmt.Sprintf(`httpd -f %s -k start -DFOREGROUND`, filepath.Join(f.Build.Application.Root, "httpd.conf"))}}},
			))
		})
	})
}
