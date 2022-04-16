package httpd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

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

func Detect(parser Parser) packit.DetectFunc {
	return func(context packit.DetectContext) (packit.DetectResult, error) {
		plan := packit.DetectResult{
			Plan: packit.BuildPlan{
				Provides: []packit.BuildPlanProvision{
					{Name: PlanDependencyHTTPD},
				},
			},
		}

		var requirements []packit.BuildPlanRequirement

		if val, ok := os.LookupEnv("BP_WEB_SERVER"); ok && val == "httpd" {
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

		if version, ok := os.LookupEnv("BP_HTTPD_VERSION"); ok {
			requirements = append(requirements, packit.BuildPlanRequirement{
				Name: PlanDependencyHTTPD,
				Metadata: BuildPlanMetadata{
					Version:       version,
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

		shouldReload, err := checkLiveReloadEnabled()
		if err != nil {
			return packit.DetectResult{}, err
		}

		if shouldReload {
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

func checkLiveReloadEnabled() (bool, error) {
	if reload, ok := os.LookupEnv("BP_LIVE_RELOAD_ENABLED"); ok {
		shouldEnableReload, err := strconv.ParseBool(reload)
		if err != nil {
			return false, fmt.Errorf("failed to parse BP_LIVE_RELOAD_ENABLED value %s: %w", reload, err)
		}
		return shouldEnableReload, nil
	}
	return false, nil
}
