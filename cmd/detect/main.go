package main

import (
	"github.com/paketo-buildpacks/httpd/httpd"
	"github.com/cloudfoundry/packit"
)

func main() {
	packit.Detect(httpd.Detect())
}
