package httpd

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/paketo-buildpacks/packit/v2/scribe"
	"github.com/paketo-buildpacks/packit/v2/servicebindings"
)

//go:generate faux --interface BindingResolver --output fakes/binding_resolver.go
type BindingResolver interface {
	Resolve(typ, provider, platformDir string) ([]servicebindings.Binding, error)
}

type GenerateHTTPDConfig struct {
	bindingResolver BindingResolver
	logger          scribe.Emitter
}

func NewGenerateHTTPDConfig(bindingResolver BindingResolver, logger scribe.Emitter) GenerateHTTPDConfig {
	return GenerateHTTPDConfig{
		bindingResolver: bindingResolver,
		logger:          logger,
	}
}

func (g GenerateHTTPDConfig) Generate(workingDir, platformPath string, buildEnvironment BuildEnvironment) error {
	g.logger.Process("Generating httpd.conf")

	t, err := template.New("httpd.conf").Parse(httpdConf)
	if err != nil {
		return err
	}

	confFile, err := os.Create(filepath.Join(workingDir, "httpd.conf"))
	if err != nil {
		return err
	}

	if buildEnvironment.WebServerRoot == "" {
		buildEnvironment.WebServerRoot = "${APP_ROOT}/public"
	} else {
		webServerRoot := buildEnvironment.WebServerRoot
		if !filepath.IsAbs(webServerRoot) {
			webServerRoot = fmt.Sprintf("${APP_ROOT}/%s", webServerRoot)
		}
		g.logger.Subprocess("Adds configuration to set web server root to '%s'", webServerRoot)
		buildEnvironment.WebServerRoot = webServerRoot
	}

	if buildEnvironment.WebServerPushStateEnabled {
		g.logger.Subprocess("Adds configuration that enables push state")
	}

	if buildEnvironment.WebServerForceHTTPS {
		g.logger.Subprocess("Adds configuration that forces https redirect")
	}

	bindings, err := g.bindingResolver.Resolve("htpasswd", "", platformPath)
	if err != nil {
		return err
	}

	if len(bindings) > 1 {
		return fmt.Errorf("failed: binding resolver found more than one binding of type 'htpasswd'")
	}

	if len(bindings) == 1 {
		if _, ok := bindings[0].Entries[".htpasswd"]; !ok {
			return fmt.Errorf("failed: binding of type 'htpasswd' does not contain required entry '.htpasswd'")
		}

		g.logger.Subprocess("Adds configuration that configured basic authentication from service binding")

		buildEnvironment.BasicAuthFile = filepath.Join(bindings[0].Path, ".htpasswd")
	}

	g.logger.Break()

	err = t.Execute(confFile, buildEnvironment)
	if err != nil {
		return err
	}

	err = confFile.Close()
	if err != nil {
		return err
	}
	return nil
}
