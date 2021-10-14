package fakes

import "sync"

type Parser struct {
	ParseVersionCall struct {
		mutex     sync.Mutex
		CallCount int
		Receives  struct {
			Path string
		}
		Returns struct {
			Version       string
			VersionSource string
			Err           error
		}
		Stub func(string) (string, string, error)
	}
}

func (f *Parser) ParseVersion(param1 string) (string, string, error) {
	f.ParseVersionCall.mutex.Lock()
	defer f.ParseVersionCall.mutex.Unlock()
	f.ParseVersionCall.CallCount++
	f.ParseVersionCall.Receives.Path = param1
	if f.ParseVersionCall.Stub != nil {
		return f.ParseVersionCall.Stub(param1)
	}
	return f.ParseVersionCall.Returns.Version, f.ParseVersionCall.Returns.VersionSource, f.ParseVersionCall.Returns.Err
}
