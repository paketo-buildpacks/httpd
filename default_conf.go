package httpd

const (
	httpdConf = `ServerRoot "${SERVER_ROOT}"

ServerName "0.0.0.0"

LoadModule mpm_event_module modules/mod_mpm_event.so
LoadModule log_config_module modules/mod_log_config.so
LoadModule mime_module modules/mod_mime.so
LoadModule dir_module modules/mod_dir.so
LoadModule authz_core_module modules/mod_authz_core.so
LoadModule unixd_module modules/mod_unixd.so
{{if or .WebServerPushStateEnabled .WebServerForceHTTPS -}}
LoadModule rewrite_module modules/mod_rewrite.so
{{end}}
{{- if .WebServerPushStateEnabled -}}
LoadModule autoindex_module modules/mod_autoindex.so
{{end}}
{{- if .BasicAuthFile -}}
LoadModule authn_core_module modules/mod_authn_core.so
LoadModule authn_file_module modules/mod_authn_file.so
LoadModule authz_host_module modules/mod_authz_host.so
LoadModule authz_user_module modules/mod_authz_user.so
LoadModule access_compat_module modules/mod_access_compat.so
LoadModule auth_basic_module modules/mod_auth_basic.so
{{end}}
TypesConfig conf/mime.types

PidFile /tmp/httpd.pid

User nobody

Listen "${PORT}"

DocumentRoot "{{.WebServerRoot}}"

DirectoryIndex index.html

ErrorLog /proc/self/fd/2

LogFormat "%h %l %u %t \"%r\" %>s %b" common
CustomLog /proc/self/fd/1 common

<Directory />
  AllowOverride None
  Require all denied
</Directory>

<Directory "{{.WebServerRoot}}">
{{- if .BasicAuthFile}}
  Require valid-user
{{- else}}
  Require all granted
{{- end}}
{{- if .WebServerPushStateEnabled}}

  Options +FollowSymLinks
  IndexIgnore */*
  RewriteEngine On
  RewriteCond %{REQUEST_FILENAME} !-f
  RewriteCond %{REQUEST_FILENAME} !-d
  RewriteRule (.*) index.html
{{- end}}
{{- if .WebServerForceHTTPS}}

  RewriteEngine On
  RewriteCond %{HTTPS} !=on
  RewriteCond %{HTTP:X-Forwarded-Proto} !https [NC]
  RewriteRule ^ https://%{HTTP_HOST}%{REQUEST_URI} [L,R=301]
{{- end}}
{{- if .BasicAuthFile}}

  AuthType Basic
  AuthName "Authentication Required"
  AuthUserFile "{{.BasicAuthFile}}"

  Order allow,deny
  Allow from all
{{- end}}
</Directory>

<Files ".ht*">
  Require all denied
</Files>`
)
