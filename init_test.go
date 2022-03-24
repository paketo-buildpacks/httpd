package httpd_test

import (
	"testing"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitHTTPD(t *testing.T) {
	suite := spec.New("httpd", spec.Report(report.Terminal{}))
	suite("Build", testBuild)
	suite("Detect", testDetect)
	suite("GenerateHTTPDConfig", testGenerateHTTPDConfig)
	suite("VersionParser", testVersionParser)
	suite.Run(t)
}
