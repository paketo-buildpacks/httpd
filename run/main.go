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
)

func main() {
	transport := cargo.NewTransport()
	dependencyService := postal.NewService(transport)
	logEmitter := scribe.NewEmitter(os.Stdout).WithLevel(os.Getenv("BP_LOG_LEVEL"))
	versionParser := httpd.NewVersionParser()
	entryResolver := draft.NewPlanner()

	packit.Run(
		httpd.Detect(versionParser),
		httpd.Build(
			entryResolver,
			dependencyService,
			chronos.DefaultClock,
			logEmitter,
		),
	)
}
