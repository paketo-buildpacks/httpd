package main

import (
	"os"

	"github.com/paketo-buildpacks/httpd"
	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/cargo"
	"github.com/paketo-buildpacks/packit/chronos"
	"github.com/paketo-buildpacks/packit/draft"
	"github.com/paketo-buildpacks/packit/postal"
)

func main() {
	transport := cargo.NewTransport()
	dependencyService := postal.NewService(transport)
	logEmitter := httpd.NewLogEmitter(os.Stdout)
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
