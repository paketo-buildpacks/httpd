package httpd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/paketo-buildpacks/packit/v2"
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

		var requirements []packit.BuildPlanRequirement

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
