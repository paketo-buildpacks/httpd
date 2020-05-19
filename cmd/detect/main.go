package main

import (
	"github.com/paketo-buildpacks/httpd/httpd"
	"github.com/paketo-buildpacks/packit"
)

func main() {
	packit.Detect(httpd.Detect())
}
