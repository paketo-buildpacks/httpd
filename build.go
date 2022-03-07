package httpd

import (
	"path/filepath"
	"time"

	"github.com/Masterminds/semver"
	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/chronos"
	"github.com/paketo-buildpacks/packit/postal"
)

//go:generate faux --interface EntryResolver --output fakes/entry_resolver.go
//go:generate faux --interface DependencyService --output fakes/dependency_service.go

type EntryResolver interface {
	Resolve(string, []packit.BuildpackPlanEntry, []interface{}) (packit.BuildpackPlanEntry, []packit.BuildpackPlanEntry)
	MergeLayerTypes(string, []packit.BuildpackPlanEntry) (launch, build bool)
}

type DependencyService interface {
	Resolve(path, name, version, stack string) (postal.Dependency, error)
	Install(dependency postal.Dependency, cnbPath, layerPath string) error
	GenerateBillOfMaterials(dependencies ...postal.Dependency) []packit.BOMEntry
}

func Build(entries EntryResolver, dependencies DependencyService, clock chronos.Clock, logger LogEmitter) packit.BuildFunc {
	return func(context packit.BuildContext) (packit.BuildResult, error) {
		logger.Title(context.BuildpackInfo)
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
		httpdLayer.Launch = launch

		if sha, ok := httpdLayer.Metadata["cache_sha"].(string); !ok || sha != dependency.SHA256 {
			logger.Process("Executing build process")

			httpdLayer, err = httpdLayer.Reset()
			if err != nil {
				return packit.BuildResult{}, err
			}
			httpdLayer.Launch = launch

			logger.Subprocess("Installing Apache HTTP Server %s", dependency.Version)
			duration, err := clock.Measure(func() error {
				return dependencies.Install(dependency, context.CNBPath, httpdLayer.Path)
			})
			if err != nil {
				return packit.BuildResult{}, err
			}
			logger.Action("Completed in %s", duration.Round(time.Millisecond))
			logger.Break()

			httpdLayer.Metadata = map[string]interface{}{
				"built_at":  clock.Now().Format(time.RFC3339Nano),
				"cache_sha": dependency.SHA256,
			}

			logger.Process("Configuring environment")
			httpdLayer.LaunchEnv.Override("APP_ROOT", context.WorkingDir)
			httpdLayer.LaunchEnv.Override("SERVER_ROOT", httpdLayer.Path)

			logger.Environment(httpdLayer.LaunchEnv)
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

		logger.LaunchProcesses(launchMetadata.Processes)

		return packit.BuildResult{
			Layers: []packit.Layer{httpdLayer},
			Launch: launchMetadata,
		}, nil
	}
}
