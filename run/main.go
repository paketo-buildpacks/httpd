package main

import (
	"fmt"
	"os"

	"github.com/paketo-buildpacks/httpd"

	"github.com/caarlos0/env/v6"
	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/cargo"
	"github.com/paketo-buildpacks/packit/v2/chronos"
	"github.com/paketo-buildpacks/packit/v2/draft"
	"github.com/paketo-buildpacks/packit/v2/postal"
	"github.com/paketo-buildpacks/packit/v2/scribe"
	"github.com/paketo-buildpacks/packit/v2/servicebindings"
)

func main() {
	transport := cargo.NewTransport()
	dependencyService := postal.NewService(transport)
	logEmitter := scribe.NewEmitter(os.Stdout).WithLevel(os.Getenv("BP_LOG_LEVEL"))
	versionParser := httpd.NewVersionParser()
	entryResolver := draft.NewPlanner()
	generateHTTPDConfig := httpd.NewGenerateHTTPDConfig(servicebindings.NewResolver(), logEmitter)

	var buildEnvironment httpd.BuildEnvironment
	err := env.Parse(&buildEnvironment)
	if err != nil {
		fmt.Fprintln(os.Stderr, fmt.Errorf("failed to parse build configuration: %w", err))
		os.Exit(1)
	}

	packit.Run(
		httpd.Detect(
			buildEnvironment,
			versionParser,
		),
		httpd.Build(
			buildEnvironment,
			entryResolver,
			dependencyService,
			generateHTTPDConfig,
			chronos.DefaultClock,
			logEmitter,
		),
	)
}
