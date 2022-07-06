package httpd_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/paketo-buildpacks/httpd"
	"github.com/paketo-buildpacks/httpd/fakes"
	"github.com/paketo-buildpacks/packit/v2/scribe"
	"github.com/paketo-buildpacks/packit/v2/servicebindings"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testGenerateHTTPDConfig(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		generateHTTPDConfig httpd.GenerateHTTPDConfig

		bindingResolver *fakes.BindingResolver

		buffer *bytes.Buffer
	)

	it.Before(func() {
		buffer = bytes.NewBuffer(nil)

		bindingResolver = &fakes.BindingResolver{}

		generateHTTPDConfig = httpd.NewGenerateHTTPDConfig(bindingResolver, scribe.NewEmitter(buffer))
	})

	context("Generate", func() {
		var (
			workingDir string
		)
		it.Before(func() {
			var err error
			workingDir, err = os.MkdirTemp("", "working-dir")
			Expect(err).NotTo(HaveOccurred())
		})

		it.After(func() {
			Expect(os.RemoveAll(workingDir)).To(Succeed())
		})

		it("create a default httpd config", func() {
			err := generateHTTPDConfig.Generate(workingDir, "platform", httpd.BuildEnvironment{})
			Expect(err).NotTo(HaveOccurred())

			Expect(bindingResolver.ResolveCall.Receives.Typ).To(Equal("htpasswd"))
			Expect(bindingResolver.ResolveCall.Receives.Provider).To(Equal(""))
			Expect(bindingResolver.ResolveCall.Receives.PlatformDir).To(Equal("platform"))

			Expect(buffer.String()).To(ContainSubstring("Generating httpd.conf"))

			contents, err := os.ReadFile(filepath.Join(workingDir, "httpd.conf"))
			Expect(err).NotTo(HaveOccurred())

			Expect(string(contents)).To(Equal(`ServerRoot "${SERVER_ROOT}"

ServerName "0.0.0.0"

LoadModule mpm_event_module modules/mod_mpm_event.so
LoadModule log_config_module modules/mod_log_config.so
LoadModule mime_module modules/mod_mime.so
LoadModule dir_module modules/mod_dir.so
LoadModule authz_core_module modules/mod_authz_core.so
LoadModule unixd_module modules/mod_unixd.so

TypesConfig conf/mime.types

PidFile /tmp/httpd.pid

User nobody

Listen "${PORT}"

DocumentRoot "${APP_ROOT}/public"

DirectoryIndex index.html

ErrorLog /proc/self/fd/2

LogFormat "%h %l %u %t \"%r\" %>s %b" common
CustomLog /proc/self/fd/1 common

<Directory />
  AllowOverride None
  Require all denied
</Directory>

<Directory "${APP_ROOT}/public">
  Require all granted
</Directory>

<Files ".ht*">
  Require all denied
</Files>`), string(contents))
		})

		context("when BP_WEB_SERVER_ROOT is set", func() {
			context("when the path given is no absolute", func() {
				it("creates a config with the adjusted DocumentRoot and Directory path", func() {
					err := generateHTTPDConfig.Generate(workingDir, "platform", httpd.BuildEnvironment{WebServerRoot: "htdocs"})
					Expect(err).NotTo(HaveOccurred())

					Expect(bindingResolver.ResolveCall.Receives.Typ).To(Equal("htpasswd"))
					Expect(bindingResolver.ResolveCall.Receives.Provider).To(Equal(""))
					Expect(bindingResolver.ResolveCall.Receives.PlatformDir).To(Equal("platform"))

					Expect(buffer.String()).To(ContainSubstring("Adds configuration to set web server root to '${APP_ROOT}/htdocs'"))

					contents, err := os.ReadFile(filepath.Join(workingDir, "httpd.conf"))
					Expect(err).NotTo(HaveOccurred())

					Expect(string(contents)).To(Equal(`ServerRoot "${SERVER_ROOT}"

ServerName "0.0.0.0"

LoadModule mpm_event_module modules/mod_mpm_event.so
LoadModule log_config_module modules/mod_log_config.so
LoadModule mime_module modules/mod_mime.so
LoadModule dir_module modules/mod_dir.so
LoadModule authz_core_module modules/mod_authz_core.so
LoadModule unixd_module modules/mod_unixd.so

TypesConfig conf/mime.types

PidFile /tmp/httpd.pid

User nobody

Listen "${PORT}"

DocumentRoot "${APP_ROOT}/htdocs"

DirectoryIndex index.html

ErrorLog /proc/self/fd/2

LogFormat "%h %l %u %t \"%r\" %>s %b" common
CustomLog /proc/self/fd/1 common

<Directory />
  AllowOverride None
  Require all denied
</Directory>

<Directory "${APP_ROOT}/htdocs">
  Require all granted
</Directory>

<Files ".ht*">
  Require all denied
</Files>`), string(contents))
				})
			})

			context("when the path given is absolute", func() {
				it("creates a config with the adjusted DocumentRoot and Directory path", func() {
					err := generateHTTPDConfig.Generate(workingDir, "platform", httpd.BuildEnvironment{WebServerRoot: "/absolute/path"})
					Expect(err).NotTo(HaveOccurred())

					Expect(bindingResolver.ResolveCall.Receives.Typ).To(Equal("htpasswd"))
					Expect(bindingResolver.ResolveCall.Receives.Provider).To(Equal(""))
					Expect(bindingResolver.ResolveCall.Receives.PlatformDir).To(Equal("platform"))

					Expect(buffer.String()).To(ContainSubstring("Adds configuration to set web server root to '/absolute/path'"))

					contents, err := os.ReadFile(filepath.Join(workingDir, "httpd.conf"))
					Expect(err).NotTo(HaveOccurred())

					Expect(string(contents)).To(Equal(`ServerRoot "${SERVER_ROOT}"

ServerName "0.0.0.0"

LoadModule mpm_event_module modules/mod_mpm_event.so
LoadModule log_config_module modules/mod_log_config.so
LoadModule mime_module modules/mod_mime.so
LoadModule dir_module modules/mod_dir.so
LoadModule authz_core_module modules/mod_authz_core.so
LoadModule unixd_module modules/mod_unixd.so

TypesConfig conf/mime.types

PidFile /tmp/httpd.pid

User nobody

Listen "${PORT}"

DocumentRoot "/absolute/path"

DirectoryIndex index.html

ErrorLog /proc/self/fd/2

LogFormat "%h %l %u %t \"%r\" %>s %b" common
CustomLog /proc/self/fd/1 common

<Directory />
  AllowOverride None
  Require all denied
</Directory>

<Directory "/absolute/path">
  Require all granted
</Directory>

<Files ".ht*">
  Require all denied
</Files>`), string(contents))
				})
			})
		})

		context("when BP_WEB_SERVER_ENABLE_PUSH_STATE is set", func() {
			it("creates a config with directices that force all routes to index.html", func() {
				err := generateHTTPDConfig.Generate(workingDir, "platform", httpd.BuildEnvironment{WebServerPushStateEnabled: true})
				Expect(err).NotTo(HaveOccurred())

				Expect(bindingResolver.ResolveCall.Receives.Typ).To(Equal("htpasswd"))
				Expect(bindingResolver.ResolveCall.Receives.Provider).To(Equal(""))
				Expect(bindingResolver.ResolveCall.Receives.PlatformDir).To(Equal("platform"))

				Expect(buffer.String()).To(ContainSubstring("Adds configuration that enables push state"))

				contents, err := os.ReadFile(filepath.Join(workingDir, "httpd.conf"))
				Expect(err).NotTo(HaveOccurred())

				Expect(string(contents)).To(Equal(`ServerRoot "${SERVER_ROOT}"

ServerName "0.0.0.0"

LoadModule mpm_event_module modules/mod_mpm_event.so
LoadModule log_config_module modules/mod_log_config.so
LoadModule mime_module modules/mod_mime.so
LoadModule dir_module modules/mod_dir.so
LoadModule authz_core_module modules/mod_authz_core.so
LoadModule unixd_module modules/mod_unixd.so
LoadModule rewrite_module modules/mod_rewrite.so
LoadModule autoindex_module modules/mod_autoindex.so

TypesConfig conf/mime.types

PidFile /tmp/httpd.pid

User nobody

Listen "${PORT}"

DocumentRoot "${APP_ROOT}/public"

DirectoryIndex index.html

ErrorLog /proc/self/fd/2

LogFormat "%h %l %u %t \"%r\" %>s %b" common
CustomLog /proc/self/fd/1 common

<Directory />
  AllowOverride None
  Require all denied
</Directory>

<Directory "${APP_ROOT}/public">
  Require all granted

  Options +FollowSymLinks
  IndexIgnore */*
  RewriteEngine On
  RewriteCond %{REQUEST_FILENAME} !-f
  RewriteCond %{REQUEST_FILENAME} !-d
  RewriteRule (.*) index.html
</Directory>

<Files ".ht*">
  Require all denied
</Files>`), string(contents))
			})
		})

		context("when BP_WEB_SERVER_FORCE_HTTPS is set", func() {
			it("creates a config with directives that force redirect to https", func() {
				err := generateHTTPDConfig.Generate(workingDir, "platform", httpd.BuildEnvironment{WebServerForceHTTPS: true})
				Expect(err).NotTo(HaveOccurred())

				Expect(bindingResolver.ResolveCall.Receives.Typ).To(Equal("htpasswd"))
				Expect(bindingResolver.ResolveCall.Receives.Provider).To(Equal(""))
				Expect(bindingResolver.ResolveCall.Receives.PlatformDir).To(Equal("platform"))

				Expect(buffer.String()).To(ContainSubstring("Adds configuration that forces https redirect"))

				contents, err := os.ReadFile(filepath.Join(workingDir, "httpd.conf"))
				Expect(err).NotTo(HaveOccurred())

				Expect(string(contents)).To(Equal(`ServerRoot "${SERVER_ROOT}"

ServerName "0.0.0.0"

LoadModule mpm_event_module modules/mod_mpm_event.so
LoadModule log_config_module modules/mod_log_config.so
LoadModule mime_module modules/mod_mime.so
LoadModule dir_module modules/mod_dir.so
LoadModule authz_core_module modules/mod_authz_core.so
LoadModule unixd_module modules/mod_unixd.so
LoadModule rewrite_module modules/mod_rewrite.so

TypesConfig conf/mime.types

PidFile /tmp/httpd.pid

User nobody

Listen "${PORT}"

DocumentRoot "${APP_ROOT}/public"

DirectoryIndex index.html

ErrorLog /proc/self/fd/2

LogFormat "%h %l %u %t \"%r\" %>s %b" common
CustomLog /proc/self/fd/1 common

<Directory />
  AllowOverride None
  Require all denied
</Directory>

<Directory "${APP_ROOT}/public">
  Require all granted

  RewriteEngine On
  RewriteCond %{HTTPS} !=on
  RewriteCond %{HTTP:X-Forwarded-Proto} !https [NC]
  RewriteRule ^ https://%{HTTP_HOST}%{REQUEST_URI} [L,R=301]
</Directory>

<Files ".ht*">
  Require all denied
</Files>`), string(contents))
			})
		})

		context("when the htpasswd service binding is set", func() {
			it.Before(func() {
				bindingResolver.ResolveCall.Returns.BindingSlice = []servicebindings.Binding{
					{
						Name: "first",
						Type: "htpasswd",
						Path: "some-binding-path",
						Entries: map[string]*servicebindings.Entry{
							".htpasswd": servicebindings.NewEntry("some-path"),
						},
					},
				}
			})

			it("creates a config with that requires basic auth", func() {
				err := generateHTTPDConfig.Generate(workingDir, "platform", httpd.BuildEnvironment{})
				Expect(err).NotTo(HaveOccurred())

				Expect(bindingResolver.ResolveCall.Receives.Typ).To(Equal("htpasswd"))
				Expect(bindingResolver.ResolveCall.Receives.Provider).To(Equal(""))
				Expect(bindingResolver.ResolveCall.Receives.PlatformDir).To(Equal("platform"))

				Expect(buffer.String()).To(ContainSubstring("Adds configuration that configured basic authentication from service binding"))

				contents, err := os.ReadFile(filepath.Join(workingDir, "httpd.conf"))
				Expect(err).NotTo(HaveOccurred())

				Expect(string(contents)).To(Equal(`ServerRoot "${SERVER_ROOT}"

ServerName "0.0.0.0"

LoadModule mpm_event_module modules/mod_mpm_event.so
LoadModule log_config_module modules/mod_log_config.so
LoadModule mime_module modules/mod_mime.so
LoadModule dir_module modules/mod_dir.so
LoadModule authz_core_module modules/mod_authz_core.so
LoadModule unixd_module modules/mod_unixd.so
LoadModule authn_core_module modules/mod_authn_core.so
LoadModule authn_file_module modules/mod_authn_file.so
LoadModule authz_host_module modules/mod_authz_host.so
LoadModule authz_user_module modules/mod_authz_user.so
LoadModule access_compat_module modules/mod_access_compat.so
LoadModule auth_basic_module modules/mod_auth_basic.so

TypesConfig conf/mime.types

PidFile /tmp/httpd.pid

User nobody

Listen "${PORT}"

DocumentRoot "${APP_ROOT}/public"

DirectoryIndex index.html

ErrorLog /proc/self/fd/2

LogFormat "%h %l %u %t \"%r\" %>s %b" common
CustomLog /proc/self/fd/1 common

<Directory />
  AllowOverride None
  Require all denied
</Directory>

<Directory "${APP_ROOT}/public">
  Require valid-user

  AuthType Basic
  AuthName "Authentication Required"
  AuthUserFile "some-binding-path/.htpasswd"

  Order allow,deny
  Allow from all
</Directory>

<Files ".ht*">
  Require all denied
</Files>`), string(contents))
			})
		})

		context("failure cases", func() {
			context("when the config file cannot be created", func() {
				it.Before(func() {
					Expect(os.Chmod(workingDir, 0000)).To(Succeed())
				})

				it.After(func() {
					Expect(os.Chmod(workingDir, os.ModePerm)).To(Succeed())
				})

				it("returns an error", func() {
					err := generateHTTPDConfig.Generate(workingDir, "platform", httpd.BuildEnvironment{})
					Expect(err).To(MatchError(ContainSubstring("permission denied")))
				})
			})

			context("when the binding resolver fails", func() {
				it.Before(func() {
					bindingResolver.ResolveCall.Returns.Error = errors.New("failed to resolve binding")
				})
				it("returns an error", func() {
					err := generateHTTPDConfig.Generate(workingDir, "platform", httpd.BuildEnvironment{})
					Expect(err).To(MatchError("failed to resolve binding"))
				})
			})

			context("when more than one binding is found", func() {
				it.Before(func() {
					bindingResolver.ResolveCall.Returns.BindingSlice = []servicebindings.Binding{
						{
							Name: "first",
							Type: "htpasswd",
							Path: "some-binding-path",
							Entries: map[string]*servicebindings.Entry{
								".htpasswd": servicebindings.NewEntry("some-path"),
							},
						},
						{
							Name: "second",
							Type: "htpasswd",
							Path: "some-binding-path",
							Entries: map[string]*servicebindings.Entry{
								".htpasswd": servicebindings.NewEntry("some-path"),
							},
						},
					}
				})
				it("returns an error", func() {
					err := generateHTTPDConfig.Generate(workingDir, "platform", httpd.BuildEnvironment{})
					Expect(err).To(MatchError("failed: binding resolver found more than one binding of type 'htpasswd'"))
				})
			})

			context("when the binding is missing the required entry", func() {
				it.Before(func() {
					bindingResolver.ResolveCall.Returns.BindingSlice = []servicebindings.Binding{
						{
							Name: "first",
							Type: "htpasswd",
							Path: "some-binding-path",
							Entries: map[string]*servicebindings.Entry{
								"wrong-entry": servicebindings.NewEntry("some-path"),
							},
						},
					}
				})
				it("returns an error", func() {
					err := generateHTTPDConfig.Generate(workingDir, "platform", httpd.BuildEnvironment{})
					Expect(err).To(MatchError("failed: binding of type 'htpasswd' does not contain required entry '.htpasswd'"))
				})
			})
		})
	})
}
