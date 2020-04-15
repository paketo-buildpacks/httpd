package httpd_test

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/paketo-buildpacks/httpd/httpd"
	"github.com/paketo-buildpacks/httpd/httpd/fakes"
	"github.com/cloudfoundry/packit"
	"github.com/cloudfoundry/packit/postal"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testBuild(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		workingDir string
		layersDir  string
		cnbPath    string
		timestamp  string

		dependencyService *fakes.DependencyService
		clock             httpd.Clock

		build packit.BuildFunc
	)

	it.Before(func() {
		var err error
		workingDir, err = ioutil.TempDir("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		layersDir, err = ioutil.TempDir("", "layers")
		Expect(err).NotTo(HaveOccurred())

		cnbPath, err = ioutil.TempDir("", "cnb-path")
		Expect(err).NotTo(HaveOccurred())

		dependencyService = &fakes.DependencyService{}
		dependencyService.ResolveCall.Returns.Dependency = postal.Dependency{
			ID:           "httpd",
			SHA256:       "some-sha",
			Source:       "some-source",
			SourceSHA256: "some-source-sha",
			Stacks:       []string{"some-stack"},
			URI:          "some-uri",
			Version:      "2.4.41",
		}

		now := time.Now()
		clock = httpd.NewClock(func() time.Time { return now })
		timestamp = now.Format(time.RFC3339Nano)

		logEmitter := httpd.NewLogEmitter(ioutil.Discard)

		build = httpd.Build(dependencyService, clock, logEmitter)
	})

	it.After(func() {
		Expect(os.RemoveAll(workingDir)).To(Succeed())
		Expect(os.RemoveAll(layersDir)).To(Succeed())
		Expect(os.RemoveAll(cnbPath)).To(Succeed())
	})

	it("builds httpd", func() {
		result, err := build(packit.BuildContext{
			WorkingDir: workingDir,
			Layers:     packit.Layers{Path: layersDir},
			CNBPath:    cnbPath,
			Stack:      "some-stack",
			Plan: packit.BuildpackPlan{
				Entries: []packit.BuildpackPlanEntry{
					{
						Name:     "httpd",
						Metadata: map[string]interface{}{"launch": true},
					},
				},
			},
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(packit.BuildResult{
			Layers: []packit.Layer{
				{
					Name:      "httpd",
					Path:      filepath.Join(layersDir, "httpd"),
					Launch:    true,
					Cache:     true,
					SharedEnv: packit.Environment{},
					BuildEnv:  packit.Environment{},
					LaunchEnv: packit.Environment{
						"APP_ROOT.override":    workingDir,
						"SERVER_ROOT.override": filepath.Join(layersDir, "httpd"),
					},
					Metadata: map[string]interface{}{
						"built_at":  timestamp,
						"cache_sha": "some-sha",
					},
				},
			},
			Processes: []packit.Process{
				{
					Type:    "web",
					Command: fmt.Sprintf("httpd -f %s -k start -DFOREGROUND", filepath.Join(workingDir, "httpd.conf")),
				},
			},
		}))

		Expect(dependencyService.ResolveCall.Receives.Path).To(Equal(filepath.Join(cnbPath, "buildpack.toml")))
		Expect(dependencyService.ResolveCall.Receives.Name).To(Equal("httpd"))
		Expect(dependencyService.ResolveCall.Receives.Version).To(Equal("*"))
		Expect(dependencyService.ResolveCall.Receives.Stack).To(Equal("some-stack"))

		Expect(dependencyService.InstallCall.Receives.Dependency).To(Equal(postal.Dependency{
			ID:           "httpd",
			SHA256:       "some-sha",
			Source:       "some-source",
			SourceSHA256: "some-source-sha",
			Stacks:       []string{"some-stack"},
			URI:          "some-uri",
			Version:      "2.4.41",
		}))
		Expect(dependencyService.InstallCall.Receives.CnbPath).To(Equal(cnbPath))
		Expect(dependencyService.InstallCall.Receives.LayerPath).To(Equal(filepath.Join(layersDir, "httpd")))
	})

	context("when the entry contains a version constraint", func() {
		it("builds httpd with that version", func() {
			result, err := build(packit.BuildContext{
				WorkingDir: workingDir,
				Layers:     packit.Layers{Path: layersDir},
				CNBPath:    cnbPath,
				Stack:      "some-stack",
				Plan: packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{
						{
							Name:     "httpd",
							Version:  "2.4.*",
							Metadata: map[string]interface{}{"launch": true},
						},
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(packit.BuildResult{
				Layers: []packit.Layer{
					{
						Name:      "httpd",
						Path:      filepath.Join(layersDir, "httpd"),
						Launch:    true,
						Cache:     true,
						SharedEnv: packit.Environment{},
						BuildEnv:  packit.Environment{},
						LaunchEnv: packit.Environment{
							"APP_ROOT.override":    workingDir,
							"SERVER_ROOT.override": filepath.Join(layersDir, "httpd"),
						},
						Metadata: map[string]interface{}{
							"built_at":  timestamp,
							"cache_sha": "some-sha",
						},
					},
				},
				Processes: []packit.Process{
					{
						Type:    "web",
						Command: fmt.Sprintf("httpd -f %s -k start -DFOREGROUND", filepath.Join(workingDir, "httpd.conf")),
					},
				},
			}))

			Expect(dependencyService.ResolveCall.Receives.Path).To(Equal(filepath.Join(cnbPath, "buildpack.toml")))
			Expect(dependencyService.ResolveCall.Receives.Name).To(Equal("httpd"))
			Expect(dependencyService.ResolveCall.Receives.Version).To(Equal("2.4.*"))
			Expect(dependencyService.ResolveCall.Receives.Stack).To(Equal("some-stack"))

			Expect(dependencyService.InstallCall.Receives.Dependency).To(Equal(postal.Dependency{
				ID:           "httpd",
				SHA256:       "some-sha",
				Source:       "some-source",
				SourceSHA256: "some-source-sha",
				Stacks:       []string{"some-stack"},
				URI:          "some-uri",
				Version:      "2.4.41",
			}))
			Expect(dependencyService.InstallCall.Receives.CnbPath).To(Equal(cnbPath))
			Expect(dependencyService.InstallCall.Receives.LayerPath).To(Equal(filepath.Join(layersDir, "httpd")))
		})
	})

	context("failure cases", func() {
		context("when the httpd layer cannot be retrieved", func() {
			it.Before(func() {
				Expect(ioutil.WriteFile(filepath.Join(layersDir, "httpd.toml"), nil, 0000)).To(Succeed())
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					WorkingDir: workingDir,
					Layers:     packit.Layers{Path: layersDir},
					CNBPath:    cnbPath,
					Stack:      "some-stack",
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{Name: "httpd", Metadata: map[string]interface{}{"launch": true}},
						},
					},
				})
				Expect(err).To(MatchError(ContainSubstring("permission denied")))
			})
		})

		context("when the dependency cannot be resolved", func() {
			it.Before(func() {
				dependencyService.ResolveCall.Returns.Error = errors.New("failed to resolve dependency")
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					WorkingDir: workingDir,
					Layers:     packit.Layers{Path: layersDir},
					CNBPath:    cnbPath,
					Stack:      "some-stack",
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{Name: "httpd", Metadata: map[string]interface{}{"launch": true}},
						},
					},
				})
				Expect(err).To(MatchError("failed to resolve dependency"))
			})
		})

		context("when the dependency cannot be installed", func() {
			it.Before(func() {
				dependencyService.InstallCall.Returns.Error = errors.New("failed to install dependency")
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					WorkingDir: workingDir,
					Layers:     packit.Layers{Path: layersDir},
					CNBPath:    cnbPath,
					Stack:      "some-stack",
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{Name: "httpd", Metadata: map[string]interface{}{"launch": true}},
						},
					},
				})
				Expect(err).To(MatchError("failed to install dependency"))
			})
		})
	})
}
