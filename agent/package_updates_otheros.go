//go:build !ios

package agent

func newIOSPackageUpdatesManager() *packageUpdatesManager { return nil }
