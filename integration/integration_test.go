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

package integration

import (
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/dagger"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	. "github.com/onsi/gomega"
)

var (
	uri string
)

func TestIntegration(t *testing.T) {
	RegisterTestingT(t)
	root, err := dagger.FindBPRoot()
	Expect(err).ToNot(HaveOccurred())
	uri, err = dagger.PackageBuildpack(root)
	Expect(err).NotTo(HaveOccurred())
	defer dagger.DeleteBuildpack(uri)
	spec.Run(t, "Integration", testIntegration, spec.Report(report.Terminal{}))
}

func testIntegration(t *testing.T, when spec.G, it spec.S) {
	var (
		app *dagger.App
		err error
	)

	it.Before(func() {
		RegisterTestingT(t)
	})

	it.After(func() {
		app.Destroy()
	})

	when("push simple app", func() {
		it("serves up staticfile", func() {
			app, err = dagger.PackBuild(filepath.Join("fixtures", "simple_app"), uri)
			Expect(err).ToNot(HaveOccurred())

			app.SetHealthCheck("", "3s", "1s")

			err = app.Start()
			Expect(err).ToNot(HaveOccurred())

			_, _, err = app.HTTPGet("/index.html")
			Expect(err).ToNot(HaveOccurred())
		})
	})

	when("the app is pushed twice", func() {
		it("uses a cached layer and doesn't run twice", func() {
			appName := "simple_app"

			app, err = dagger.PackBuildNamedImage(appName, filepath.Join("fixtures", appName), uri)
			Expect(err).ToNot(HaveOccurred())

			app.SetHealthCheck("", "3s", "1s")

			err = app.Start()
			Expect(err).ToNot(HaveOccurred())

			Expect(app.BuildLogs()).To(MatchRegexp("Apache HTTP Server .*: Contributing to layer"))

			app, err = dagger.PackBuildNamedImage(appName, filepath.Join("fixtures", appName), uri)
			Expect(err).ToNot(HaveOccurred())

			Expect(app.BuildLogs()).To(MatchRegexp("Apache HTTP Server .*: Reusing cached layer"))
			Expect(app.BuildLogs()).NotTo(MatchRegexp("Apache HTTP Server .*: Contributing to layer"))

			Expect(app.Start()).To(Succeed())
		})
	})
}
