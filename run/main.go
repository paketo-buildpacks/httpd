package main

import (
	"os"

	"github.com/paketo-buildpacks/httpd"
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

	packit.Run(
		httpd.Detect(versionParser),
		httpd.Build(
			entryResolver,
			dependencyService,
			generateHTTPDConfig,
			chronos.DefaultClock,
			logEmitter,
		),
	)
}
