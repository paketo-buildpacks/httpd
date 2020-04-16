package main

import (
	"os"
	"time"

	"github.com/paketo-buildpacks/httpd/httpd"
	"github.com/cloudfoundry/packit"
	"github.com/cloudfoundry/packit/cargo"
	"github.com/cloudfoundry/packit/postal"
)

func main() {
	transport := cargo.NewTransport()
	dependencyService := postal.NewService(transport)
	clock := httpd.NewClock(time.Now)
	logEmitter := httpd.NewLogEmitter(os.Stdout)

	packit.Build(httpd.Build(dependencyService, clock, logEmitter))
}
