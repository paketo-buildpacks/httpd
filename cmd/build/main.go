package main

import (
	"os"
	"time"

	"github.com/paketo-buildpacks/httpd/httpd"
	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/cargo"
	"github.com/paketo-buildpacks/packit/postal"
)

func main() {
	transport := cargo.NewTransport()
	dependencyService := postal.NewService(transport)
	clock := httpd.NewClock(time.Now)
	logEmitter := httpd.NewLogEmitter(os.Stdout)

	packit.Build(httpd.Build(dependencyService, clock, logEmitter))
}
