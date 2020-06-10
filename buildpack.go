package httpd

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

type Buildpack struct {
	HTTPD BuildpackHTTPD `yaml:"httpd"`
}

type BuildpackHTTPD struct {
	Version string `yaml:"version"`
}

func ParseBuildpack(path string) (Buildpack, error) {
	file, err := os.Open(path)
	if err != nil {
		return Buildpack{}, fmt.Errorf("failed to parse buildpack.yml: %w", err)
	}
	defer file.Close()

	var buildpack Buildpack
	err = yaml.NewDecoder(file).Decode(&buildpack)
	if err != nil {
		return Buildpack{}, fmt.Errorf("failed to parse buildpack.yml: %w", err)
	}

	return buildpack, nil
}
