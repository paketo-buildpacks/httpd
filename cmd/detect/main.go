package main

import (
	"fmt"
	"github.com/buildpack/libbuildpack/buildplan"
	"github.com/cloudfoundry/httpd-cnb/httpd"
	"github.com/cloudfoundry/libcfbuildpack/layers"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/libcfbuildpack/detect"
)

func main() {
	detectionContext, err := detect.DefaultDetect()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to run detection: %s", err)
		os.Exit(101)
	}

	code, err := runDetect(detectionContext)
	if err != nil {
		detectionContext.Logger.Info(err.Error())
	}

	os.Exit(code)
}

func runDetect(context detect.Detect) (int, error) {
	exists, err := layers.FileExists(filepath.Join(context.Application.Root, "httpd.conf"))
	if err != nil {
		return context.Fail(), err
	}

	if !exists {
		return context.Fail(), fmt.Errorf("unable to find httpd.conf")
	}

	return context.Pass(buildplan.BuildPlan{
		httpd.Dependency: buildplan.Dependency{
			Metadata: buildplan.Metadata{"launch": true},
		},
	})
}
