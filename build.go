package httpd

import (
	"fmt"
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

		if sha, ok := httpdLayer.Metadata["cache_sha"].(string); !ok || sha != dependency.SHA256 {
			logger.Process("Executing build process")

			httpdLayer, err = httpdLayer.Reset()
			if err != nil {
				return packit.BuildResult{}, err
			}
			httpdLayer.Launch, _ = entries.MergeLayerTypes("httpd", context.Plan.Entries)

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

		return packit.BuildResult{
			Layers: []packit.Layer{httpdLayer},
			Launch: packit.LaunchMetadata{
				Processes: []packit.Process{
					{
						Type:    "web",
						Command: fmt.Sprintf("httpd -f %s -k start -DFOREGROUND", filepath.Join(context.WorkingDir, "httpd.conf")),
					},
				},
			},
		}, nil
	}
}
