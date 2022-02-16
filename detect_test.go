package httpd_test

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/paketo-buildpacks/httpd"
	"github.com/paketo-buildpacks/httpd/fakes"
	"github.com/paketo-buildpacks/packit"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testDetect(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		parser *fakes.Parser

		workingDir string
		detect     packit.DetectFunc
	)

	it.Before(func() {
		var err error
		workingDir, err = ioutil.TempDir("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		parser = &fakes.Parser{}
		parser.ParseVersionCall.Returns.Version = "some-version"
		parser.ParseVersionCall.Returns.VersionSource = "some-version-source"

		detect = httpd.Detect(parser)
	})

	it.After(func() {
		Expect(os.RemoveAll(workingDir)).To(Succeed())
	})

	it("returns a DetectResult that provides httpd", func() {
		result, err := detect(packit.DetectContext{
			WorkingDir: workingDir,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(packit.DetectResult{
			Plan: packit.BuildPlan{
				Provides: []packit.BuildPlanProvision{
					{Name: httpd.PlanDependencyHTTPD},
				},
			},
		}))

		Expect(parser.ParseVersionCall.CallCount).To(Equal(0))
	})

	context("when there is an httpd.conf file in the workspace", func() {
		it.Before(func() {
			_, err := os.Create(filepath.Join(workingDir, "httpd.conf"))
			Expect(err).NotTo(HaveOccurred())
		})

		it("returns a DetectResult that provides and required httpd", func() {
			result, err := detect(packit.DetectContext{
				WorkingDir: workingDir,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(packit.DetectResult{
				Plan: packit.BuildPlan{
					Provides: []packit.BuildPlanProvision{
						{Name: httpd.PlanDependencyHTTPD},
					},
					Requires: []packit.BuildPlanRequirement{
						{
							Name: httpd.PlanDependencyHTTPD,
							Metadata: httpd.BuildPlanMetadata{
								Version:       "some-version",
								VersionSource: "some-version-source",
								Launch:        true,
							},
						},
					},
				},
			}))

			Expect(parser.ParseVersionCall.Receives.Path).To(Equal(filepath.Join(workingDir, "buildpack.yml")))
		})

		context("and BP_LIVE_RELOAD_ENABLED=true in the build environment", func() {
			it.Before(func() {
				os.Setenv("BP_LIVE_RELOAD_ENABLED", "true")
			})

			it.After(func() {
				os.Unsetenv("BP_LIVE_RELOAD_ENABLED")
			})

			it("requires watchexec at launch time", func() {
				result, err := detect(packit.DetectContext{
					WorkingDir: workingDir,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(result.Plan.Requires).To(Equal([]packit.BuildPlanRequirement{
					{
						Name: httpd.PlanDependencyHTTPD,
						Metadata: httpd.BuildPlanMetadata{
							Version:       "some-version",
							VersionSource: "some-version-source",
							Launch:        true,
						},
					},
					{
						Name: "watchexec",
						Metadata: map[string]interface{}{
							"launch": true,
						},
					},
				},
				))
			})
		})
	})

	context("BP_HTTPD_VERSION is set", func() {
		it.Before(func() {
			_, err := os.Create(filepath.Join(workingDir, "httpd.conf"))
			Expect(err).NotTo(HaveOccurred())
			Expect(os.Setenv("BP_HTTPD_VERSION", "env-var-version")).To(Succeed())
		})

		it.After(func() {
			err := os.Remove(filepath.Join(workingDir, "httpd.conf"))
			Expect(err).NotTo(HaveOccurred())
			Expect(os.Unsetenv("BP_HTTPD_VERSION")).To(Succeed())
		})

		it("returns a DetectResult that required specified version of httpd", func() {
			result, err := detect(packit.DetectContext{
				WorkingDir: workingDir,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(packit.DetectResult{
				Plan: packit.BuildPlan{
					Provides: []packit.BuildPlanProvision{
						{Name: httpd.PlanDependencyHTTPD},
					},
					Requires: []packit.BuildPlanRequirement{
						{
							Name: httpd.PlanDependencyHTTPD,
							Metadata: httpd.BuildPlanMetadata{
								Version:       "env-var-version",
								VersionSource: "BP_HTTPD_VERSION",
								Launch:        true,
							},
						},
						{
							Name: httpd.PlanDependencyHTTPD,
							Metadata: httpd.BuildPlanMetadata{
								Version:       "some-version",
								VersionSource: "some-version-source",
								Launch:        true,
							},
						},
					},
				},
			}))
		})
	})

	context("failure cases", func() {
		it.Before(func() {
			_, err := os.Create(filepath.Join(workingDir, "httpd.conf"))
			Expect(err).NotTo(HaveOccurred())
		})

		context("when ParseVersion fails", func() {
			it.Before(func() {
				parser.ParseVersionCall.Returns.Err = errors.New("failed to parse version")
			})

			it("returns an error", func() {
				_, err := detect(packit.DetectContext{WorkingDir: workingDir})
				Expect(err).To(MatchError("failed to parse version"))
			})
		})

		context("when BP_LIVE_RELOAD_ENABLED is set to an invalid value", func() {
			it.Before(func() {
				os.Setenv("BP_LIVE_RELOAD_ENABLED", "not-a-bool")
			})

			it.After(func() {
				os.Unsetenv("BP_LIVE_RELOAD_ENABLED")
			})

			it("returns an error", func() {
				_, err := detect(packit.DetectContext{
					WorkingDir: workingDir,
				})
				Expect(err).To(MatchError(ContainSubstring("failed to parse BP_LIVE_RELOAD_ENABLED value not-a-bool")))
			})
		})
	})
}
