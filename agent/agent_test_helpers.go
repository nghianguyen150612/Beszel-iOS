//go:build testing

package agent

// TESTING ONLY: GetConnectionManager is a helper function to get the connection manager for testing.
func (a *Agent) GetConnectionManager() *ConnectionManager {
	return a.connectionManager
}

// ConnectionStateForTesting uses the production synchronized state reader.
func (a *Agent) ConnectionStateForTesting() ConnectionState { return a.connectionManager.getState() }

// CancelForTesting requests normal manager shutdown; callers must join Start.
func (a *Agent) CancelForTesting() { a.connectionManager.cancel() }
