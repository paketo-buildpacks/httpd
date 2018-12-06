package httpd

import (
	"github.com/cloudfoundry/libcfbuildpack/build"
	"github.com/cloudfoundry/libcfbuildpack/layers"
)

const Dependency = "httpd"

type Contributor struct {
	launchContribution bool
	launchLayer        layers.Layers
	httpdLayer         layers.DependencyLayer
}

func NewContributor(context build.Build) (Contributor, bool, error) {
	plan, wantDependency := context.BuildPlan[Dependency]
	if !wantDependency {
		return Contributor{}, false, nil
	}

	deps, err := context.Buildpack.Dependencies()
	if err != nil {
		return Contributor{}, false, err
	}

	dep, err := deps.Best(Dependency, plan.Version, context.Stack)
	if err != nil {
		return Contributor{}, false, err
	}

	contributor := Contributor{
		launchLayer: context.Layers,
		httpdLayer:  context.Layers.DependencyLayer(dep),
	}

	if _, ok := plan.Metadata["launch"]; ok {
		contributor.launchContribution = true
	}

	return contributor, true, nil
}

func (c Contributor) Contribute() error {
	return c.httpdLayer.Contribute(func(artifact string, layer layers.DependencyLayer) error {
		layer.Logger.SubsequentLine("Expanding to %s", layer.Root)
		if err := layers.ExtractTarGz(artifact, layer.Root, 1); err != nil {
			return err
		}

		return c.launchLayer.WriteMetadata(layers.Metadata{
			Processes: []layers.Process{{"web", `apachectl -f "httpd.conf" -k start -DFOREGROUND`}},
		})
	}, c.flags()...)
}

func (n Contributor) flags() []layers.Flag {
	var flags []layers.Flag

	if n.launchContribution {
		flags = append(flags, layers.Launch)
	}

	return flags
}
