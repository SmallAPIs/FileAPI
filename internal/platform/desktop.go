// Package platform provides OS-specific desktop integration (open URL/app).
package platform

// Desktop opens URLs and applications on the host operating system.
type Desktop interface {
	OpenURL(url string) error
	OpenApp(nameOrPath string) error
}

// New returns the platform-specific Desktop implementation.
func New() Desktop {
	return newDesktop()
}
