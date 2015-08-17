// +build !clamav

package bot

import "github.com/Sirupsen/logrus"

type clamEngine struct {
}

// newClamEngine creates an dummy engine
func newClamEngine() (*clamEngine, error) {
	return &clamEngine{}, nil
}

// scan without ClamAV returns empty result
func (ce *clamEngine) scan(filename string, b []byte) (string, error) {
	logrus.Debug("ClamAV is not configured to run")
	return "", nil
}

func (ce *clamEngine) close() {
}
