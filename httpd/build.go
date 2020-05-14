package httpd

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/postal"
)

//go:generate faux --interface DependencyService --output fakes/dependency_service.go
type DependencyService interface {
	Resolve(path, name, version, stack string) (postal.Dependency, error)
	Install(dependency postal.Dependency, cnbPath, layerPath string) error
}

func Build(dependencies DependencyService, clock Clock, logger LogEmitter) packit.BuildFunc {
	return func(context packit.BuildContext) (packit.BuildResult, error) {
		logger.Title(context.BuildpackInfo)

		logger.Process("Resolving Apache HTTP Server version")
		logger.Candidates(context.Plan.Entries)

		entry := context.Plan.Entries[0]

		httpdLayer, err := context.Layers.Get("httpd", packit.CacheLayer)
		if err != nil {
			return packit.BuildResult{}, err
		}

		httpdLayer.Launch = entry.Metadata["launch"] == true

		version := "*"
		if entry.Version != "" {
			version = entry.Version
		}

		dependency, err := dependencies.Resolve(filepath.Join(context.CNBPath, "buildpack.toml"), "httpd", version, context.Stack)
		if err != nil {
			return packit.BuildResult{}, err
		}

		logger.SelectedDependency(entry, dependency.Version)

		if sha, ok := httpdLayer.Metadata["cache_sha"].(string); !ok || sha != dependency.SHA256 {
			logger.Break()
			logger.Process("Executing build process")

			err = httpdLayer.Reset()
			if err != nil {
				return packit.BuildResult{}, err
			}

			logger.Subprocess("Installing Apache HTTP Server %s", dependency.Version)
			then := clock.Now()
			err = dependencies.Install(dependency, context.CNBPath, httpdLayer.Path)
			if err != nil {
				return packit.BuildResult{}, err
			}
			logger.Action("Completed in %s", time.Since(then).Round(time.Millisecond))
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
			Processes: []packit.Process{
				{
					Type:    "web",
					Command: fmt.Sprintf("httpd -f %s -k start -DFOREGROUND", filepath.Join(context.WorkingDir, "httpd.conf")),
				},
			},
		}, nil
	}
}
