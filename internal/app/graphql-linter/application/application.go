package application

import (
	"runtime/debug"
)

type Executor interface {
	Run()
}

type Execute struct{}

func (e Execute) Run() {}

func VersionString() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		return info.Main.Version
	}

	return "(unknown)"
}
