package httpd

import (
	"path/filepath"

	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/fs"
)

const PlanDependencyHTTPD = "httpd"

//go:generate faux --interface Parser --output fakes/parser.go
type Parser interface {
	ParseVersion(path string) (version, versionSource string, err error)
}

type BuildPlanMetadata struct {
	Version       string `toml:"version,omitempty"`
	VersionSource string `toml:"version-source,omitempty"`
	Launch        bool   `toml:"launch"`
}

func Detect(buildEnvironment BuildEnvironment, parser Parser) packit.DetectFunc {
	return func(context packit.DetectContext) (packit.DetectResult, error) {
		plan := packit.DetectResult{
			Plan: packit.BuildPlan{
				Provides: []packit.BuildPlanProvision{
					{Name: PlanDependencyHTTPD},
				},
			},
		}

		var requirements []packit.BuildPlanRequirement

		if buildEnvironment.WebServer == "httpd" {
			requirements = append(requirements, packit.BuildPlanRequirement{
				Name: PlanDependencyHTTPD,
				Metadata: BuildPlanMetadata{
					Launch: true,
				},
			})
			plan.Plan.Requires = requirements
		}

		exists, err := fs.Exists(filepath.Join(context.WorkingDir, "httpd.conf"))
		if err != nil {
			return packit.DetectResult{}, err
		}

		if !exists {
			return plan, nil
		}

		if buildEnvironment.HTTPDVersion != "" {
			requirements = append(requirements, packit.BuildPlanRequirement{
				Name: PlanDependencyHTTPD,
				Metadata: BuildPlanMetadata{
					Version:       buildEnvironment.HTTPDVersion,
					VersionSource: "BP_HTTPD_VERSION",
					Launch:        true,
				},
			})
		}

		version, versionSource, err := parser.ParseVersion(filepath.Join(context.WorkingDir, "buildpack.yml"))
		if err != nil {
			return packit.DetectResult{}, err
		}

		if version != "" {
			requirements = append(requirements, packit.BuildPlanRequirement{
				Name: PlanDependencyHTTPD,
				Metadata: BuildPlanMetadata{
					Version:       version,
					VersionSource: versionSource,
					Launch:        true,
				},
			})
			plan.Plan.Requires = requirements
		}

		if buildEnvironment.Reload {
			requirements = append(requirements, packit.BuildPlanRequirement{
				Name: "watchexec",
				Metadata: map[string]interface{}{
					"launch": true,
				},
			})
			plan.Plan.Requires = requirements
		}

		return plan, nil
	}
}
