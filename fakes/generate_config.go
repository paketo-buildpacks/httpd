package fakes

import "sync"

type GenerateConfig struct {
	GenerateCall struct {
		mutex     sync.Mutex
		CallCount int
		Receives  struct {
			WorkingDir   string
			PlatformPath string
		}
		Returns struct {
			Error error
		}
		Stub func(string, string) error
	}
}

func (f *GenerateConfig) Generate(param1 string, param2 string) error {
	f.GenerateCall.mutex.Lock()
	defer f.GenerateCall.mutex.Unlock()
	f.GenerateCall.CallCount++
	f.GenerateCall.Receives.WorkingDir = param1
	f.GenerateCall.Receives.PlatformPath = param2
	if f.GenerateCall.Stub != nil {
		return f.GenerateCall.Stub(param1, param2)
	}
	return f.GenerateCall.Returns.Error
}
