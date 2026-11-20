//go:build testing

package health

// SetPathsForTesting redirects both health markers for tests in this module.
func SetPathsForTesting(livenessPath, connectionPath string) func() {
	return SetHealthPathsForTesting([]string{livenessPath}, connectionPath)
}

func SetHealthPathsForTesting(livenessPaths []string, connectionPath string) func() {
	originalHealthFile, originalHealthFiles, originalConnectionFile := healthFile, healthFiles, connectionFile
	healthFiles = livenessPaths
	if len(livenessPaths) > 0 {
		healthFile = livenessPaths[0]
	}
	connectionFile = connectionPath
	return func() {
		healthFile, healthFiles, connectionFile = originalHealthFile, originalHealthFiles, originalConnectionFile
	}
}
