package httpd_test

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/paketo-buildpacks/httpd"
	"github.com/paketo-buildpacks/httpd/fakes"
	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/chronos"
	"github.com/paketo-buildpacks/packit/v2/sbom"

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
		sbomGenerator     *fakes.SBOMGenerator

		buffer *bytes.Buffer

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
			SHA256:       "some-sha", //nolint:staticcheck
			Source:       "some-source",
			SourceSHA256: "some-source-sha", //nolint:staticcheck
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

		sbomGenerator = &fakes.SBOMGenerator{}
		sbomGenerator.GenerateFromDependencyCall.Returns.SBOM = sbom.SBOM{}

		buffer = bytes.NewBuffer(nil)

		build = httpd.Build(httpd.BuildEnvironment{}, entryResolver, dependencyService, generateConfig, sbomGenerator, chronos.DefaultClock, scribe.NewEmitter(buffer))
	})

	it.After(func() {
		Expect(os.RemoveAll(workingDir)).To(Succeed())
		Expect(os.RemoveAll(layersDir)).To(Succeed())
		Expect(os.RemoveAll(cnbPath)).To(Succeed())
	})

	it("builds httpd", func() {
		result, err := build(packit.BuildContext{
			BuildpackInfo: packit.BuildpackInfo{
				Name:        "Some Buildpack",
				Version:     "1.2.3",
				SBOMFormats: []string{sbom.CycloneDXFormat, sbom.SPDXFormat},
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

		Expect(layer.SBOM.Formats()).To(HaveLen(2))
		cdx := layer.SBOM.Formats()[0]
		spdx := layer.SBOM.Formats()[1]

		Expect(cdx.Extension).To(Equal("cdx.json"))
		content, err := io.ReadAll(cdx.Content)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(content)).To(MatchJSON(`{
			"$schema": "http://cyclonedx.org/schema/bom-1.3.schema.json",
			"bomFormat": "CycloneDX",
			"metadata": {
				"tools": [
					{
						"name": "",
						"vendor": "anchore"
					}
				]
			},
			"specVersion": "1.3",
			"version": 1
		}`))

		Expect(spdx.Extension).To(Equal("spdx.json"))
		content, err = io.ReadAll(spdx.Content)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(content)).To(MatchJSON(`{
			"SPDXID": "SPDXRef-DOCUMENT",
			"creationInfo": {
				"created": "0001-01-01T00:00:00Z",
				"creators": [
					"Organization: Anchore, Inc",
					"Tool: -"
				],
				"licenseListVersion": "3.27"
			},
            "packages": [
                {
                  "SPDXID": "SPDXRef-DocumentRoot-Unknown-",
                  "copyrightText": "NOASSERTION",
                  "downloadLocation": "NOASSERTION",
                  "filesAnalyzed": false,
                  "licenseConcluded": "NOASSERTION",
                  "licenseDeclared": "NOASSERTION",
                  "name": "",
                  "supplier": "NOASSERTION"
                }
              ],
			"dataLicense": "CC0-1.0",
            "documentNamespace": "https://paketo.io/unknown-source-type/unknown-33ef57ff-45c2-53a8-8899-1c2b7e94d0dd",
			"name": "unknown",
			"relationships": [
				{
				    "relatedSpdxElement": "SPDXRef-DocumentRoot-Unknown-",
					"relationshipType": "DESCRIBES",
					"spdxElementId": "SPDXRef-DOCUMENT"
				}
			],
			"spdxVersion": "SPDX-2.2"
		}`))

		Expect(dependencyService.ResolveCall.Receives.Path).To(Equal(filepath.Join(cnbPath, "buildpack.toml")))
		Expect(dependencyService.ResolveCall.Receives.Name).To(Equal("httpd"))
		Expect(dependencyService.ResolveCall.Receives.Version).To(Equal("some-env-var-version"))
		Expect(dependencyService.ResolveCall.Receives.Stack).To(Equal("some-stack"))

		Expect(dependencyService.DeliverCall.Receives.Dependency).To(Equal(postal.Dependency{
			ID:           "httpd",
			SHA256:       "some-sha", //nolint:staticcheck
			Source:       "some-source",
			SourceSHA256: "some-source-sha", //nolint:staticcheck
			Stacks:       []string{"some-stack"},
			URI:          "some-uri",
			Version:      "some-env-var-version",
		}))
		Expect(dependencyService.DeliverCall.Receives.CnbPath).To(Equal(cnbPath))
		Expect(dependencyService.DeliverCall.Receives.LayerPath).To(Equal(filepath.Join(layersDir, "httpd")))
		Expect(dependencyService.DeliverCall.Receives.PlatformPath).To(Equal("platform"))

		Expect(generateConfig.GenerateCall.CallCount).To(Equal(0))

		Expect(sbomGenerator.GenerateFromDependencyCall.Receives.Dependency).To(Equal(postal.Dependency{
			ID:           "httpd",
			SHA256:       "some-sha", //nolint:staticcheck
			Source:       "some-source",
			SourceSHA256: "some-source-sha", //nolint:staticcheck
			Stacks:       []string{"some-stack"},
			URI:          "some-uri",
			Version:      "some-env-var-version",
		}))
		Expect(sbomGenerator.GenerateFromDependencyCall.Receives.Dir).To(Equal(filepath.Join(layersDir, "httpd")))
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
				SHA256:       "some-sha", //nolint:staticcheck
				Source:       "some-source",
				SourceSHA256: "some-source-sha", //nolint:staticcheck
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
				SHA256:       "some-sha", //nolint:staticcheck
				Source:       "some-source",
				SourceSHA256: "some-source-sha", //nolint:staticcheck
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
			build = httpd.Build(
				httpd.BuildEnvironment{
					WebServer: "httpd",
				},
				entryResolver,
				dependencyService,
				generateConfig,
				sbomGenerator,
				chronos.DefaultClock,
				scribe.NewEmitter(buffer),
			)
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
			Expect(generateConfig.GenerateCall.Receives.BuildEnvironment).To(Equal(httpd.BuildEnvironment{
				WebServer: "httpd",
			}))
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
			build = httpd.Build(
				httpd.BuildEnvironment{
					Reload: true,
				},
				entryResolver,
				dependencyService,
				generateConfig,
				sbomGenerator,
				chronos.DefaultClock,
				scribe.NewEmitter(buffer),
			)
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

		context("when generating the config file fails", func() {
			it.Before(func() {
				generateConfig.GenerateCall.Returns.Error = errors.New("failed to generate config file")

				build = httpd.Build(
					httpd.BuildEnvironment{
						WebServer: "httpd",
					},
					entryResolver,
					dependencyService,
					generateConfig,
					sbomGenerator,
					chronos.DefaultClock,
					scribe.NewEmitter(buffer),
				)
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

		context("when generating the SBOM returns an error", func() {
			it.Before(func() {
				sbomGenerator.GenerateFromDependencyCall.Returns.Error = errors.New("failed to generate SBOM")
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					CNBPath: cnbPath,
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{Name: "httpd"},
						},
					},
					Layers: packit.Layers{Path: layersDir},
					Stack:  "some-stack",
				})
				Expect(err).To(MatchError(ContainSubstring("failed to generate SBOM")))
			})
		})

		context("when formatting the SBOM returns an error", func() {
			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					BuildpackInfo: packit.BuildpackInfo{SBOMFormats: []string{"random-format"}},
					CNBPath:       cnbPath,
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{Name: "httpd"},
						},
					},
					Layers: packit.Layers{Path: layersDir},
					Stack:  "some-stack",
				})
				Expect(err).To(MatchError("unsupported SBOM format: 'random-format'"))
			})
		})
	})
}
