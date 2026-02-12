package update

// Checker polls the update server for new firmware versions.
type Checker struct {
	serverURL string
	devMode   bool
}

// NewChecker creates an update checker.
func NewChecker(serverURL string, devMode bool) *Checker {
	return &Checker{serverURL: serverURL, devMode: devMode}
}
