package httpd_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/paketo-buildpacks/httpd"
	"github.com/paketo-buildpacks/httpd/fakes"
	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/chronos"

	//nolint Ignore SA1019, informed usage of deprecated package
	"github.com/paketo-buildpacks/packit/v2/paketosbom"
	"github.com/paketo-buildpacks/packit/v2/postal"
	"github.com/paketo-buildpacks/packit/v2/scribe"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testBuild(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		workingDir string
		layersDir  string
		cnbPath    string

		entryResolver     *fakes.EntryResolver
		dependencyService *fakes.DependencyService
		generateConfig    *fakes.GenerateConfig
		buffer            *bytes.Buffer

		build packit.BuildFunc
	)

	it.Before(func() {
		var err error
		workingDir, err = os.MkdirTemp("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		layersDir, err = os.MkdirTemp("", "layers")
		Expect(err).NotTo(HaveOccurred())

		cnbPath, err = os.MkdirTemp("", "cnb-path")
		Expect(err).NotTo(HaveOccurred())

		entryResolver = &fakes.EntryResolver{}
		entryResolver.ResolveCall.Returns.BuildpackPlanEntry = packit.BuildpackPlanEntry{
			Name: "http",
			Metadata: map[string]interface{}{
				"version-source": "BP_HTTPD_VERSION",
				"version":        "some-env-var-version",
				"launch":         true,
			},
		}
		entryResolver.MergeLayerTypesCall.Returns.Launch = true

		dependencyService = &fakes.DependencyService{}
		dependencyService.ResolveCall.Returns.Dependency = postal.Dependency{
			ID:           "httpd",
			SHA256:       "some-sha",
			Source:       "some-source",
			SourceSHA256: "some-source-sha",
			Stacks:       []string{"some-stack"},
			URI:          "some-uri",
			Version:      "some-env-var-version",
		}
		dependencyService.GenerateBillOfMaterialsCall.Returns.BOMEntrySlice = []packit.BOMEntry{
			{
				Name: "httpd",
				Metadata: paketosbom.BOMMetadata{
					Version: "httpd-dependency-version",
					Checksum: paketosbom.BOMChecksum{
						Algorithm: paketosbom.SHA256,
						Hash:      "httpd-dependency-sha",
					},
					URI: "httpd-dependency-uri",
				},
			},
		}

		generateConfig = &fakes.GenerateConfig{}

		buffer = bytes.NewBuffer(nil)

		build = httpd.Build(entryResolver, dependencyService, generateConfig, chronos.DefaultClock, scribe.NewEmitter(buffer))
	})

	it.After(func() {
		Expect(os.RemoveAll(workingDir)).To(Succeed())
		Expect(os.RemoveAll(layersDir)).To(Succeed())
		Expect(os.RemoveAll(cnbPath)).To(Succeed())
	})

	it("builds httpd", func() {
		result, err := build(packit.BuildContext{
			BuildpackInfo: packit.BuildpackInfo{
				Name:    "Some Buildpack",
				Version: "1.2.3",
			},
			WorkingDir: workingDir,
			Layers:     packit.Layers{Path: layersDir},
			CNBPath:    cnbPath,
			Stack:      "some-stack",
			Platform:   packit.Platform{Path: "platform"},
			Plan: packit.BuildpackPlan{
				Entries: []packit.BuildpackPlanEntry{
					{
						Name: "httpd",
						Metadata: map[string]interface{}{
							"version-source": "BP_HTTPD_VERSION",
							"version":        "some-env-var-version",
							"launch":         true,
						},
					},
				},
			},
		})
		Expect(err).NotTo(HaveOccurred())

		Expect(result.Layers).To(HaveLen(1))
		layer := result.Layers[0]

		Expect(layer.Name).To(Equal("httpd"))
		Expect(layer.Path).To(Equal(filepath.Join(layersDir, "httpd")))
		Expect(layer.Build).To(BeFalse())
		Expect(layer.Cache).To(BeFalse())
		Expect(layer.Launch).To(BeTrue())
		Expect(layer.LaunchEnv).To(Equal(packit.Environment{
			"APP_ROOT.override":    workingDir,
			"SERVER_ROOT.override": filepath.Join(layersDir, "httpd"),
		}))
		Expect(layer.Metadata).To(Equal(map[string]interface{}{
			"cache_sha": "some-sha",
		}))

		Expect(result.Launch.BOM).To(Equal([]packit.BOMEntry{
			{
				Name: "httpd",
				Metadata: paketosbom.BOMMetadata{
					Version: "httpd-dependency-version",
					Checksum: paketosbom.BOMChecksum{
						Algorithm: paketosbom.SHA256,
						Hash:      "httpd-dependency-sha",
					},
					URI: "httpd-dependency-uri",
				},
			},
		}))

		Expect(result.Launch.Processes).To(Equal([]packit.Process{
			{
				Type:    "web",
				Command: "httpd",
				Args: []string{
					"-f",
					filepath.Join(workingDir, "httpd.conf"),
					"-k",
					"start",
					"-DFOREGROUND",
				},
				Default: true,
				Direct:  true,
			},
		}))

		Expect(dependencyService.ResolveCall.Receives.Path).To(Equal(filepath.Join(cnbPath, "buildpack.toml")))
		Expect(dependencyService.ResolveCall.Receives.Name).To(Equal("httpd"))
		Expect(dependencyService.ResolveCall.Receives.Version).To(Equal("some-env-var-version"))
		Expect(dependencyService.ResolveCall.Receives.Stack).To(Equal("some-stack"))

		Expect(dependencyService.DeliverCall.Receives.Dependency).To(Equal(postal.Dependency{
			ID:           "httpd",
			SHA256:       "some-sha",
			Source:       "some-source",
			SourceSHA256: "some-source-sha",
			Stacks:       []string{"some-stack"},
			URI:          "some-uri",
			Version:      "some-env-var-version",
		}))
		Expect(dependencyService.DeliverCall.Receives.CnbPath).To(Equal(cnbPath))
		Expect(dependencyService.DeliverCall.Receives.LayerPath).To(Equal(filepath.Join(layersDir, "httpd")))
		Expect(dependencyService.DeliverCall.Receives.PlatformPath).To(Equal("platform"))

		Expect(generateConfig.GenerateCall.CallCount).To(Equal(0))
	})

	context("when the entry contains a version constraint", func() {
		it.Before(func() {
			entryResolver.ResolveCall.Returns.BuildpackPlanEntry = packit.BuildpackPlanEntry{
				Name: "http",
				Metadata: map[string]interface{}{
					"version-source": "BP_HTTPD_VERSION",
					"version":        "2.4.*",
					"launch":         true,
				},
			}

			dependencyService.ResolveCall.Returns.Dependency = postal.Dependency{
				ID:           "httpd",
				SHA256:       "some-sha",
				Source:       "some-source",
				SourceSHA256: "some-source-sha",
				Stacks:       []string{"some-stack"},
				URI:          "some-uri",
				Version:      "2.4.41",
			}
		})

		it("builds httpd with that version", func() {
			result, err := build(packit.BuildContext{
				BuildpackInfo: packit.BuildpackInfo{
					Name:    "Some Buildpack",
					Version: "1.2.3",
				},
				WorkingDir: workingDir,
				Layers:     packit.Layers{Path: layersDir},
				CNBPath:    cnbPath,
				Stack:      "some-stack",
				Platform:   packit.Platform{Path: "platform"},
				Plan: packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{
						{
							Name: "httpd",
							Metadata: map[string]interface{}{
								"version-source": "BP_HTTPD_VERSION",
								"version":        "2.4.*",
								"launch":         true,
							},
						},
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(result.Layers).To(HaveLen(1))
			layer := result.Layers[0]

			Expect(layer.Name).To(Equal("httpd"))
			Expect(layer.Path).To(Equal(filepath.Join(layersDir, "httpd")))
			Expect(layer.Build).To(BeFalse())
			Expect(layer.Cache).To(BeFalse())
			Expect(layer.Launch).To(BeTrue())
			Expect(layer.LaunchEnv).To(Equal(packit.Environment{
				"APP_ROOT.override":    workingDir,
				"SERVER_ROOT.override": filepath.Join(layersDir, "httpd"),
			}))
			Expect(layer.Metadata).To(Equal(map[string]interface{}{
				"cache_sha": "some-sha",
			}))

			Expect(result.Launch.BOM).To(Equal([]packit.BOMEntry{
				{
					Name: "httpd",
					Metadata: paketosbom.BOMMetadata{
						Version: "httpd-dependency-version",
						Checksum: paketosbom.BOMChecksum{
							Algorithm: paketosbom.SHA256,
							Hash:      "httpd-dependency-sha",
						},
						URI: "httpd-dependency-uri",
					},
				},
			}))

			Expect(result.Launch.Processes).To(Equal([]packit.Process{
				{
					Type:    "web",
					Command: "httpd",
					Args: []string{
						"-f",
						filepath.Join(workingDir, "httpd.conf"),
						"-k",
						"start",
						"-DFOREGROUND",
					},
					Default: true,
					Direct:  true,
				},
			}))

			Expect(dependencyService.ResolveCall.Receives.Path).To(Equal(filepath.Join(cnbPath, "buildpack.toml")))
			Expect(dependencyService.ResolveCall.Receives.Name).To(Equal("httpd"))
			Expect(dependencyService.ResolveCall.Receives.Version).To(Equal("2.4.*"))
			Expect(dependencyService.ResolveCall.Receives.Stack).To(Equal("some-stack"))

			Expect(dependencyService.DeliverCall.Receives.Dependency).To(Equal(postal.Dependency{
				ID:           "httpd",
				SHA256:       "some-sha",
				Source:       "some-source",
				SourceSHA256: "some-source-sha",
				Stacks:       []string{"some-stack"},
				URI:          "some-uri",
				Version:      "2.4.41",
			}))
			Expect(dependencyService.DeliverCall.Receives.CnbPath).To(Equal(cnbPath))
			Expect(dependencyService.DeliverCall.Receives.LayerPath).To(Equal(filepath.Join(layersDir, "httpd")))
			Expect(dependencyService.DeliverCall.Receives.PlatformPath).To(Equal("platform"))
		})
	})

	context("when the version source is buildpack.yml", func() {
		it.Before(func() {
			entryResolver.ResolveCall.Returns.BuildpackPlanEntry = packit.BuildpackPlanEntry{
				Name: "http",
				Metadata: map[string]interface{}{
					"version-source": "buildpack.yml",
					"version":        "some-bp-yml-version",
					"launch":         true,
				},
			}
		})

		it("builds httpd with that version", func() {
			_, err := build(packit.BuildContext{
				BuildpackInfo: packit.BuildpackInfo{
					Name:    "Some Buildpack",
					Version: "1.2.3",
				},
				WorkingDir: workingDir,
				Layers:     packit.Layers{Path: layersDir},
				CNBPath:    cnbPath,
				Stack:      "some-stack",
				Plan: packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{
						{
							Name: "httpd",
							Metadata: map[string]interface{}{
								"version-source": "buildpack.yml",
								"version":        "some-bp-yml-version",
								"launch":         true,
							},
						},
					},
				},
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(dependencyService.ResolveCall.Receives.Path).To(Equal(filepath.Join(cnbPath, "buildpack.toml")))
			Expect(dependencyService.ResolveCall.Receives.Name).To(Equal("httpd"))
			Expect(dependencyService.ResolveCall.Receives.Version).To(Equal("some-bp-yml-version"))
			Expect(dependencyService.ResolveCall.Receives.Stack).To(Equal("some-stack"))

			Expect(buffer.String()).To(ContainSubstring("WARNING: Setting the server version through buildpack.yml will be deprecated soon in Apache HTTP Server Buildpack v2.0.0"))
			Expect(buffer.String()).To(ContainSubstring("Please specify the version through the $BP_HTTPD_VERSION environment variable instead. See docs for more information."))
		})
	})

	context("when BP_WEB_SERVER=httpd", func() {
		it.Before(func() {
			os.Setenv("BP_WEB_SERVER", "httpd")
		})

		it.After(func() {
			os.Unsetenv("BP_WEB_SERVER")
		})

		it("generates a httpd.conf", func() {
			_, err := build(packit.BuildContext{
				BuildpackInfo: packit.BuildpackInfo{
					Name:    "Some Buildpack",
					Version: "1.2.3",
				},
				WorkingDir: workingDir,
				Layers:     packit.Layers{Path: layersDir},
				CNBPath:    cnbPath,
				Stack:      "some-stack",
				Platform:   packit.Platform{Path: "platform"},
				Plan: packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{
						{
							Name: "httpd",
							Metadata: map[string]interface{}{
								"launch": true,
							},
						},
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(generateConfig.GenerateCall.Receives.WorkingDir).To(Equal(workingDir))
			Expect(generateConfig.GenerateCall.Receives.PlatformPath).To(Equal("platform"))
		})
	})

	context("when the layer metadata contains a cache match", func() {
		it.Before(func() {
			err := os.WriteFile(filepath.Join(layersDir, "httpd.toml"),
				[]byte("[metadata]\ncache_sha = \"some-sha\"\n"), 0600)
			Expect(err).NotTo(HaveOccurred())

			entryResolver.MergeLayerTypesCall.Returns.Launch = true
		})

		it("reuses the layer", func() {
			result, err := build(packit.BuildContext{
				BuildpackInfo: packit.BuildpackInfo{
					Name:    "Some Buildpack",
					Version: "1.2.3",
				},
				WorkingDir: workingDir,
				Layers:     packit.Layers{Path: layersDir},
				CNBPath:    cnbPath,
				Stack:      "some-stack",
				Plan: packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{
						{
							Name: "httpd",
							Metadata: map[string]interface{}{
								"version-source": "BP_HTTPD_VERSION",
								"version":        "some-env-var-version",
								"launch":         true,
							},
						},
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(result.Layers).To(HaveLen(1))
			layer := result.Layers[0]

			Expect(layer.Name).To(Equal("httpd"))
			Expect(layer.Path).To(Equal(filepath.Join(layersDir, "httpd")))
			Expect(layer.Build).To(BeFalse())
			Expect(layer.Cache).To(BeFalse())
			Expect(layer.Launch).To(BeTrue())

			Expect(result.Launch.BOM).To(Equal([]packit.BOMEntry{
				{
					Name: "httpd",
					Metadata: paketosbom.BOMMetadata{
						Version: "httpd-dependency-version",
						Checksum: paketosbom.BOMChecksum{
							Algorithm: paketosbom.SHA256,
							Hash:      "httpd-dependency-sha",
						},
						URI: "httpd-dependency-uri",
					},
				},
			}))

			Expect(result.Launch.Processes).To(Equal([]packit.Process{
				{
					Type:    "web",
					Command: "httpd",
					Args: []string{
						"-f",
						filepath.Join(workingDir, "httpd.conf"),
						"-k",
						"start",
						"-DFOREGROUND",
					},
					Default: true,
					Direct:  true,
				},
			}))

			Expect(dependencyService.DeliverCall.CallCount).To(Equal(0))
		})
	})

	context("when BP_LIVE_RELOAD_ENABLED=true in the build environment", func() {
		it.Before(func() {
			os.Setenv("BP_LIVE_RELOAD_ENABLED", "true")
		})

		it.After(func() {
			os.Unsetenv("BP_LIVE_RELOAD_ENABLED")
		})

		it("uses watchexec to set the start command", func() {
			result, err := build(packit.BuildContext{
				BuildpackInfo: packit.BuildpackInfo{
					Name:    "Some Buildpack",
					Version: "1.2.3",
				},
				WorkingDir: workingDir,
				Layers:     packit.Layers{Path: layersDir},
				CNBPath:    cnbPath,
				Stack:      "some-stack",
				Plan: packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{
						{
							Name: "httpd",
							Metadata: map[string]interface{}{
								"version-source": "BP_HTTPD_VERSION",
								"version":        "some-env-var-version",
								"launch":         true,
							},
						},
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Launch.Processes).To(Equal([]packit.Process{
				{
					Type:    "web",
					Command: "watchexec",
					Args: []string{
						"--restart",
						"--watch", workingDir,
						"--shell", "none",
						"--",
						"httpd",
						"-f",
						filepath.Join(workingDir, "httpd.conf"),
						"-k",
						"start",
						"-DFOREGROUND",
					},
					Default: true,
					Direct:  true,
				},
				{
					Type:    "no-reload",
					Command: "httpd",
					Args: []string{
						"-f",
						filepath.Join(workingDir, "httpd.conf"),
						"-k",
						"start",
						"-DFOREGROUND",
					},
					Direct: true,
				},
			}))
		})
	})

	context("failure cases", func() {
		context("when the httpd layer cannot be retrieved", func() {
			it.Before(func() {
				Expect(os.WriteFile(filepath.Join(layersDir, "httpd.toml"), nil, 0000)).To(Succeed())
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					WorkingDir: workingDir,
					Layers:     packit.Layers{Path: layersDir},
					CNBPath:    cnbPath,
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
				})
				Expect(err).To(MatchError("failed to resolve dependency"))
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
				_, err := build(packit.BuildContext{
					WorkingDir: workingDir,
					Layers:     packit.Layers{Path: layersDir},
					CNBPath:    cnbPath,
				})
				Expect(err).To(MatchError(ContainSubstring("failed to parse BP_LIVE_RELOAD_ENABLED value not-a-bool")))
			})
		})

		context("when generating the config file fails", func() {
			it.Before(func() {
				generateConfig.GenerateCall.Returns.Error = errors.New("failed to generate config file")

				os.Setenv("BP_WEB_SERVER", "httpd")
			})

			it.After(func() {
				os.Unsetenv("BP_WEB_SERVER")
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					WorkingDir: workingDir,
					Layers:     packit.Layers{Path: layersDir},
					CNBPath:    cnbPath,
				})
				Expect(err).To(MatchError("failed to generate config file"))
			})
		})

		context("when the dependency cannot be installed", func() {
			it.Before(func() {
				dependencyService.DeliverCall.Returns.Error = errors.New("failed to install dependency")
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					WorkingDir: workingDir,
					Layers:     packit.Layers{Path: layersDir},
					CNBPath:    cnbPath,
				})
				Expect(err).To(MatchError("failed to install dependency"))
			})
		})
	})
}
