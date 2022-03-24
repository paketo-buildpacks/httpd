package httpd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"text/template"

	"github.com/paketo-buildpacks/packit/v2/scribe"
)

type GenerateHTTPDConfig struct {
	logger scribe.Emitter
}

type configOptions struct {
	WebServerRoot string
	PushState     bool
}

func NewGenerateHTTPDConfig(logger scribe.Emitter) GenerateHTTPDConfig {
	return GenerateHTTPDConfig{
		logger: logger,
	}
}

func (g GenerateHTTPDConfig) Generate(workingDir string) error {
	t, err := template.New("httpd.conf").Parse(httpdConf)
	if err != nil {
		return err
	}

	confFile, err := os.Create(filepath.Join(workingDir, "httpd.conf"))
	if err != nil {
		return err
	}

	confOptions := configOptions{
		WebServerRoot: "public",
	}

	if val, ok := os.LookupEnv("BP_WEB_SERVER_ROOT"); ok {
		confOptions.WebServerRoot = val
	}

	confOptions.PushState, err = checkEnvironemntVariableTruthy("BP_WEB_SERVER_ENABLE_PUSH_STATE")
	if err != nil {
		return err
	}

	err = t.Execute(confFile, confOptions)
	if err != nil {
		return err
	}

	err = confFile.Close()
	if err != nil {
		return err
	}
	return nil
}

const (
	httpdConf = `ServerRoot "${SERVER_ROOT}"

ServerName "0.0.0.0"

LoadModule mpm_event_module modules/mod_mpm_event.so
LoadModule log_config_module modules/mod_log_config.so
LoadModule mime_module modules/mod_mime.so
LoadModule dir_module modules/mod_dir.so
LoadModule authz_core_module modules/mod_authz_core.so
LoadModule unixd_module modules/mod_unixd.so
{{if .PushState -}}
LoadModule rewrite_module modules/mod_rewrite.so
LoadModule autoindex_module modules/mod_autoindex.so
{{end}}
TypesConfig conf/mime.types

PidFile logs/httpd.pid

User nobody

Listen "${PORT}"

DocumentRoot "${APP_ROOT}/{{.WebServerRoot}}"

DirectoryIndex index.html

ErrorLog logs/error_log

LogFormat "%h %l %u %t \"%r\" %>s %b" common
CustomLog logs/access_log common

<Directory />
  AllowOverride None
  Require all denied
</Directory>

<Directory "${APP_ROOT}/{{.WebServerRoot}}">
  Require all granted
{{- if .PushState}}

  Options +FollowSymLinks
  IndexIgnore */*
  RewriteEngine On
  RewriteCond %{REQUEST_FILENAME} !-f
  RewriteCond %{REQUEST_FILENAME} !-d
  RewriteRule (.*) index.html
{{- end}}
</Directory>`
)

func checkEnvironemntVariableTruthy(env string) (bool, error) {
	if reload, ok := os.LookupEnv(env); ok {
		shouldEnableReload, err := strconv.ParseBool(reload)
		if err != nil {
			return false, fmt.Errorf("failed to parse %s value %s: %w", env, reload, err)
		}
		return shouldEnableReload, nil
	}
	return false, nil
}
