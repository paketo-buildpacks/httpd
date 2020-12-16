package httpd

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/paketo-buildpacks/packit"
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

func Detect(parser Parser) packit.DetectFunc {
	return func(context packit.DetectContext) (packit.DetectResult, error) {
		plan := packit.DetectResult{
			Plan: packit.BuildPlan{
				Provides: []packit.BuildPlanProvision{
					{Name: PlanDependencyHTTPD},
				},
			},
		}

		_, err := os.Stat(filepath.Join(context.WorkingDir, "httpd.conf"))
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return plan, nil
			}
			return packit.DetectResult{}, err
		}

		version, versionSource, err := parser.ParseVersion(filepath.Join(context.WorkingDir, "buildpack.yml"))
		if err != nil {
			return packit.DetectResult{}, err
		}

		plan.Plan.Requires = []packit.BuildPlanRequirement{
			{
				Name: PlanDependencyHTTPD,
				Metadata: BuildPlanMetadata{
					Version:       version,
					VersionSource: versionSource,
					Launch:        true,
				},
			},
		}

		return plan, nil
	}
}
