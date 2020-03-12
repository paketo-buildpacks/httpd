package main

import (
	"github.com/cloudfoundry/httpd-cnb/httpd"
	"github.com/cloudfoundry/packit"
)

func main() {
	packit.Detect(httpd.Detect())
}
