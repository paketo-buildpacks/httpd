package fakes

import (
	"sync"

	"github.com/paketo-buildpacks/httpd"
)

type GenerateConfig struct {
	GenerateCall struct {
		mutex     sync.Mutex
		CallCount int
		Receives  struct {
			WorkingDir       string
			PlatformPath     string
			BuildEnvironment httpd.BuildEnvironment
		}
		Returns struct {
			Error error
		}
		Stub func(string, string, httpd.BuildEnvironment) error
	}
}

func (f *GenerateConfig) Generate(param1 string, param2 string, param3 httpd.BuildEnvironment) error {
	f.GenerateCall.mutex.Lock()
	defer f.GenerateCall.mutex.Unlock()
	f.GenerateCall.CallCount++
	f.GenerateCall.Receives.WorkingDir = param1
	f.GenerateCall.Receives.PlatformPath = param2
	f.GenerateCall.Receives.BuildEnvironment = param3
	if f.GenerateCall.Stub != nil {
		return f.GenerateCall.Stub(param1, param2, param3)
	}
	return f.GenerateCall.Returns.Error
}
