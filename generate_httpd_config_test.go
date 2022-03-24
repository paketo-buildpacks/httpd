package httpd_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/paketo-buildpacks/httpd"
	"github.com/paketo-buildpacks/packit/v2/scribe"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testGenerateHTTPDConfig(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		generateHTTPDConfig httpd.GenerateHTTPDConfig

		buffer *bytes.Buffer
	)

	it.Before(func() {
		buffer = bytes.NewBuffer(nil)

		generateHTTPDConfig = httpd.NewGenerateHTTPDConfig(scribe.NewEmitter(buffer))
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
			err := generateHTTPDConfig.Generate(workingDir)
			Expect(err).NotTo(HaveOccurred())

			contents, err := os.ReadFile(filepath.Join(workingDir, "httpd.conf"))
			Expect(err).NotTo(HaveOccurred())

			Expect(string(contents)).To(Equal(`ServerRoot "${SERVER_ROOT}"

LoadModule mpm_event_module modules/mod_mpm_event.so
LoadModule log_config_module modules/mod_log_config.so
LoadModule mime_module modules/mod_mime.so
LoadModule dir_module modules/mod_dir.so
LoadModule authz_core_module modules/mod_authz_core.so
LoadModule unixd_module modules/mod_unixd.so

TypesConfig conf/mime.types

PidFile logs/httpd.pid

User nobody

Listen "${PORT}"

DocumentRoot "${APP_ROOT}/public"

DirectoryIndex index.html

ErrorLog logs/error_log

LogFormat "%h %l %u %t \"%r\" %>s %b" common
CustomLog logs/access_log common

<Directory />
  AllowOverride None
  Require all denied
</Directory>

<Directory "${APP_ROOT}/public">
  Require all granted
</Directory>`))
		})

		context("when BP_WEB_SERVER_ROOT is set", func() {
			it.Before(func() {
				os.Setenv("BP_WEB_SERVER_ROOT", "htdocs")
			})

			it.After(func() {
				os.Unsetenv("BP_WEB_SERVER_ROOT")
			})

			it("creates a config with the adjusted DocumentRoot and Directory path", func() {
				err := generateHTTPDConfig.Generate(workingDir)
				Expect(err).NotTo(HaveOccurred())

				contents, err := os.ReadFile(filepath.Join(workingDir, "httpd.conf"))
				Expect(err).NotTo(HaveOccurred())

				Expect(string(contents)).To(Equal(`ServerRoot "${SERVER_ROOT}"

LoadModule mpm_event_module modules/mod_mpm_event.so
LoadModule log_config_module modules/mod_log_config.so
LoadModule mime_module modules/mod_mime.so
LoadModule dir_module modules/mod_dir.so
LoadModule authz_core_module modules/mod_authz_core.so
LoadModule unixd_module modules/mod_unixd.so

TypesConfig conf/mime.types

PidFile logs/httpd.pid

User nobody

Listen "${PORT}"

DocumentRoot "${APP_ROOT}/htdocs"

DirectoryIndex index.html

ErrorLog logs/error_log

LogFormat "%h %l %u %t \"%r\" %>s %b" common
CustomLog logs/access_log common

<Directory />
  AllowOverride None
  Require all denied
</Directory>

<Directory "${APP_ROOT}/htdocs">
  Require all granted
</Directory>`))

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
					err := generateHTTPDConfig.Generate(workingDir)
					Expect(err).To(MatchError(ContainSubstring("permission denied")))
				})
			})
		})
	})
}
