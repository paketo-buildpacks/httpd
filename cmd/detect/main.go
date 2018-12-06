package main

import (
	"fmt"
	"github.com/buildpack/libbuildpack/buildplan"
	"github.com/cloudfoundry/httpd-cnb/httpd"
	"github.com/cloudfoundry/libcfbuildpack/layers"
	"gopkg.in/yaml.v2"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/libcfbuildpack/detect"
)

type BuildpackYAML struct {
	Config httpd.Config `yaml:"httpd"`
}

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

	// TODO : we should add functionality to libbuildpack or libcfbuildpack to load buildpack.yml files as that is the generic way to configure them
	buildpackYAML, configFile := BuildpackYAML{}, filepath.Join(context.Application.Root, "buildpack.yml")
	if exists, err := layers.FileExists(configFile); err != nil {
		return context.Fail(), err
	} else if exists {
		file, err := os.Open(configFile)
		if err != nil {
			return context.Fail(), err
		}
		defer file.Close()

		err = yaml.NewDecoder(file).Decode(&buildpackYAML)
		if err != nil {
			return context.Fail(), err
		}
	}

	return context.Pass(buildplan.BuildPlan{
		httpd.Dependency: buildplan.Dependency{
			Version:  buildpackYAML.Config.Version,
			Metadata: buildplan.Metadata{"launch": true},
		},
	})
}
