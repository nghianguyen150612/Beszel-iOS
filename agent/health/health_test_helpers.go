//go:build testing

package health

// SetPathsForTesting redirects both health markers for tests in this module.
func SetPathsForTesting(livenessPath, connectionPath string) func() {
	originalHealthFile, originalConnectionFile := healthFile, connectionFile
	healthFile, connectionFile = livenessPath, connectionPath
	return func() { healthFile, connectionFile = originalHealthFile, originalConnectionFile }
}
