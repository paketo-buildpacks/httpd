package httpd

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

type VersionParser struct{}

func NewVersionParser() VersionParser {
	return VersionParser{}
}

func (v VersionParser) ParseVersion(path string) (string, string, error) {
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "*", "", nil
		}
		return "", "", fmt.Errorf("failed to parse buildpack.yml: %w", err)
	}

	defer file.Close()

	var buildpack struct {
		Httpd struct {
			Version string `yaml:"version"`
		} `yaml:"httpd"`
	}
	err = yaml.NewDecoder(file).Decode(&buildpack)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse buildpack.yml: %w", err)
	}

	if buildpack.Httpd.Version == "" {
		return "*", "", nil
	}

	return buildpack.Httpd.Version, "buildpack.yml", nil
}
