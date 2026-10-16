//go:build testing

package agent

// TESTING ONLY: GetConnectionManager is a helper function to get the connection manager for testing.
func (a *Agent) GetConnectionManager() *ConnectionManager {
	return a.connectionManager
}

// StateForTesting observes connection state under the production reader lock.
func (c *ConnectionManager) StateForTesting() ConnectionState { return c.getState() }

// CancelForTesting stops the manager through its ordinary Start-loop cleanup.
func (c *ConnectionManager) CancelForTesting() { c.cancel() }
