package integration_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/cloudfoundry/dagger"
	"github.com/cloudfoundry/packit/pexec"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	. "github.com/onsi/gomega"
)

var uri string

func TestIntegration(t *testing.T) {
	var Expect = NewWithT(t).Expect

	root, err := dagger.FindBPRoot()
	Expect(err).ToNot(HaveOccurred())

	uri, err = dagger.PackageBuildpack(root)
	Expect(err).NotTo(HaveOccurred())

	defer dagger.DeleteBuildpack(uri)

	suite := spec.New("Integration", spec.Report(report.Terminal{}))
	suite("Caching", testCaching)
	suite("SimpleApp", testSimpleApp)
	suite("Logging", testLogging)
	suite.Run(t)
}

func GetGitVersion() (string, error) {
	stdout := bytes.NewBuffer(nil)
	git := pexec.NewExecutable("git")

	err := git.Execute(pexec.Execution{
		Args:   []string{"describe", "--abbrev=0", "--tags"},
		Stdout: stdout,
	})
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(stdout.String()), nil
}
