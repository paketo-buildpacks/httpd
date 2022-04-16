package httpd

import (
	"os"
	"path/filepath"
	"time"

	"github.com/Masterminds/semver"
	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/chronos"
	"github.com/paketo-buildpacks/packit/v2/postal"
	"github.com/paketo-buildpacks/packit/v2/scribe"
)

//go:generate faux --interface EntryResolver --output fakes/entry_resolver.go
type EntryResolver interface {
	Resolve(name string, entries []packit.BuildpackPlanEntry, priorites []interface{}) (packit.BuildpackPlanEntry, []packit.BuildpackPlanEntry)
	MergeLayerTypes(name string, entries []packit.BuildpackPlanEntry) (launch, build bool)
}

//go:generate faux --interface DependencyService --output fakes/dependency_service.go
type DependencyService interface {
	Resolve(path, name, version, stack string) (postal.Dependency, error)
	Deliver(dependency postal.Dependency, cnbPath, layerPath, platformPath string) error
	GenerateBillOfMaterials(dependencies ...postal.Dependency) []packit.BOMEntry
}

//go:generate faux --interface GenerateConfig --output fakes/generate_config.go
type GenerateConfig interface {
	Generate(workingDir, platformPath string) error
}

func Build(entries EntryResolver, dependencies DependencyService, generateConfig GenerateConfig, clock chronos.Clock, logger scribe.Emitter) packit.BuildFunc {
	return func(context packit.BuildContext) (packit.BuildResult, error) {
		logger.Title("%s %s", context.BuildpackInfo.Name, context.BuildpackInfo.Version)
		logger.Process("Resolving Apache HTTP Server version")

		priorities := []interface{}{
			"BP_HTTPD_VERSION",
			"buildpack.yml",
		}
		entry, sortedEntries := entries.Resolve("httpd", context.Plan.Entries, priorities)
		logger.Candidates(sortedEntries)

		httpdLayer, err := context.Layers.Get("httpd")
		if err != nil {
			return packit.BuildResult{}, err
		}

		version, ok := entry.Metadata["version"].(string)
		if !ok {
			version = "*"
		}

		dependency, err := dependencies.Resolve(filepath.Join(context.CNBPath, "buildpack.toml"), "httpd", version, context.Stack)
		if err != nil {
			return packit.BuildResult{}, err
		}

		logger.SelectedDependency(entry, dependency, clock.Now())

		source, _ := entry.Metadata["version-source"].(string)
		if source == "buildpack.yml" {
			nextMajorVersion := semver.MustParse(context.BuildpackInfo.Version).IncMajor()
			logger.Subprocess("WARNING: Setting the server version through buildpack.yml will be deprecated soon in Apache HTTP Server Buildpack v%s.", nextMajorVersion.String())
			logger.Subprocess("Please specify the version through the $BP_HTTPD_VERSION environment variable instead. See docs for more information.")
			logger.Break()
		}

		launch, _ := entries.MergeLayerTypes("httpd", context.Plan.Entries)
		bom := dependencies.GenerateBillOfMaterials(dependency)

		var launchMetadata packit.LaunchMetadata
		if launch {
			launchMetadata.BOM = bom
		}

		command := "httpd"
		args := []string{
			"-f",
			filepath.Join(context.WorkingDir, "httpd.conf"),
			"-k",
			"start",
			"-DFOREGROUND",
		}
		launchMetadata.Processes = []packit.Process{
			{
				Type:    "web",
				Command: command,
				Args:    args,
				Default: true,
				Direct:  true,
			},
		}

		shouldReload, err := checkLiveReloadEnabled()
		if err != nil {
			return packit.BuildResult{}, err
		}

		if shouldReload {
			launchMetadata.Processes = []packit.Process{
				{
					Type:    "web",
					Command: "watchexec",
					Args: append([]string{
						"--restart",
						"--watch", context.WorkingDir,
						"--shell", "none",
						"--",
						command,
					}, args...),
					Default: true,
					Direct:  true,
				},
				{
					Type:    "no-reload",
					Command: command,
					Args:    args,
					Direct:  true,
				},
			}
		}

		if val, ok := os.LookupEnv("BP_WEB_SERVER"); ok && val == "httpd" {
			err = generateConfig.Generate(context.WorkingDir, context.Platform.Path)
			if err != nil {
				return packit.BuildResult{}, err
			}
		}

		cachedSHA, ok := httpdLayer.Metadata["cache_sha"].(string)
		if ok && cachedSHA == dependency.SHA256 {
			logger.Process("Reusing cached layer %s", httpdLayer.Path)
			logger.Break()

			httpdLayer.Launch = launch

			logger.LaunchProcesses(launchMetadata.Processes)

			return packit.BuildResult{
				Layers: []packit.Layer{httpdLayer},
				Launch: launchMetadata,
			}, nil
		}

		logger.Process("Executing build process")

		httpdLayer, err = httpdLayer.Reset()
		if err != nil {
			return packit.BuildResult{}, err
		}
		httpdLayer.Launch = launch

		logger.Subprocess("Installing Apache HTTP Server %s", dependency.Version)
		duration, err := clock.Measure(func() error {
			return dependencies.Deliver(dependency, context.CNBPath, httpdLayer.Path, context.Platform.Path)
		})
		if err != nil {
			return packit.BuildResult{}, err
		}
		logger.Action("Completed in %s", duration.Round(time.Millisecond))
		logger.Break()

		httpdLayer.Metadata = map[string]interface{}{
			"cache_sha": dependency.SHA256,
		}

		httpdLayer.LaunchEnv.Override("APP_ROOT", context.WorkingDir)
		httpdLayer.LaunchEnv.Override("SERVER_ROOT", httpdLayer.Path)

		logger.EnvironmentVariables(httpdLayer)

		logger.LaunchProcesses(launchMetadata.Processes)

		return packit.BuildResult{
			Layers: []packit.Layer{httpdLayer},
			Launch: launchMetadata,
		}, nil
	}
}
