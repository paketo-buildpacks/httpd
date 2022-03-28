package httpd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
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

type configOptions struct {
	WebServerRoot string
	PushState     bool
	ForceHTTPS    bool
	HtpasswdPath  string
}

func NewGenerateHTTPDConfig(bindingResolver BindingResolver, logger scribe.Emitter) GenerateHTTPDConfig {
	return GenerateHTTPDConfig{
		bindingResolver: bindingResolver,
		logger:          logger,
	}
}

func (g GenerateHTTPDConfig) Generate(workingDir, platformPath string) error {
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

	confOptions.ForceHTTPS, err = checkEnvironemntVariableTruthy("BP_WEB_SERVER_FORCE_HTTPS")
	if err != nil {
		return err
	}

	bindings, err := g.bindingResolver.Resolve("htpasswd", "", platformPath)
	if err != nil {
		return err
	}

	if len(bindings) > 1 {
		return fmt.Errorf("failed: binding resolver found more than one binding of type 'htpasswd'")
	}

	if len(bindings) == 1 {
		// p.logs.Process("Loading service binding of type '%s'", typ)

		if _, ok := bindings[0].Entries[".htpasswd"]; !ok {
			return fmt.Errorf("failed: binding of type 'htpasswd' does not contain required entry '.htpasswd'")
		}

		confOptions.HtpasswdPath = filepath.Join(bindings[0].Path, ".htpasswd")
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
{{if or .PushState .ForceHTTPS -}}
LoadModule rewrite_module modules/mod_rewrite.so
{{end}}
{{- if .PushState -}}
LoadModule autoindex_module modules/mod_autoindex.so
{{end}}
{{- if .HtpasswdPath -}}
LoadModule authn_core_module modules/mod_authn_core.so
LoadModule authn_file_module modules/mod_authn_file.so
LoadModule authz_host_module modules/mod_authz_host.so
LoadModule authz_user_module modules/mod_authz_user.so
LoadModule access_compat_module modules/mod_access_compat.so
LoadModule auth_basic_module modules/mod_auth_basic.so
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
{{- if .HtpasswdPath}}
  Require valid-user
{{- else}}
  Require all granted
{{- end}}
{{- if .PushState}}

  Options +FollowSymLinks
  IndexIgnore */*
  RewriteEngine On
  RewriteCond %{REQUEST_FILENAME} !-f
  RewriteCond %{REQUEST_FILENAME} !-d
  RewriteRule (.*) index.html
{{- end}}
{{- if .ForceHTTPS}}

  RewriteEngine On
  RewriteCond %{HTTPS} !=on
  RewriteCond %{HTTP:X-Forwarded-Proto} !https [NC]
  RewriteRule ^ https://%{HTTP_HOST}%{REQUEST_URI} [L,R=301]
{{- end}}
{{- if .HtpasswdPath}}

  AuthType Basic
  AuthName "Authentication Required"
  AuthUserFile "{{.HtpasswdPath}}"

  Order allow,deny
  Allow from all
{{- end}}
</Directory>

<Files ".ht*">
  Require all denied
</Files>`
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
