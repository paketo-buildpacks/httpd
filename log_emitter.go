package httpd

import (
	"io"

	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/scribe"
)

type LogEmitter struct {
	scribe.Emitter
}

func NewLogEmitter(output io.Writer) LogEmitter {
	return LogEmitter{
		Emitter: scribe.NewEmitter(output),
	}
}

func (e LogEmitter) Title(info packit.BuildpackInfo) {
	e.Logger.Title("%s %s", info.Name, info.Version)
}

func (e LogEmitter) Environment(environment packit.Environment) {
	e.Logger.Subprocess("%s", scribe.NewFormattedMapFromEnvironment(environment))
}
