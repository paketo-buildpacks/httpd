package httpd

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/paketo-buildpacks/packit"
)

const PlanDependencyHTTPD = "httpd"

type BuildPlanMetadata struct {
	VersionSource string `toml:"version-source,omitempty"`
	Launch        bool   `toml:"launch"`
}

func Detect() packit.DetectFunc {
	return func(context packit.DetectContext) (packit.DetectResult, error) {
		var requirements []packit.BuildPlanRequirement

		_, err := os.Stat(filepath.Join(context.WorkingDir, "httpd.conf"))
		if err == nil {
			requirements = append(requirements, packit.BuildPlanRequirement{
				Name: PlanDependencyHTTPD,
				Metadata: BuildPlanMetadata{
					Launch: true,
				},
			})
		}
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return packit.DetectResult{}, err
		}

		buildpack, err := ParseBuildpack(filepath.Join(context.WorkingDir, "buildpack.yml"))
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return packit.DetectResult{}, err
		}

		if buildpack.HTTPD.Version != "" {
			if len(requirements) != 1 {
				return packit.DetectResult{}, errors.New("failed to detect: buildpack.yml specifies a version, but httpd.conf is missing")
			}

			requirements[0].Version = buildpack.HTTPD.Version
			requirements[0].Metadata = BuildPlanMetadata{
				VersionSource: "buildpack.yml",
				Launch:        true,
			}
		}

		return packit.DetectResult{
			Plan: packit.BuildPlan{
				Provides: []packit.BuildPlanProvision{{Name: PlanDependencyHTTPD}},
				Requires: requirements,
			},
		}, nil
	}
}
