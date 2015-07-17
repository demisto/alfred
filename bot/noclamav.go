// +build !clamav

package bot

import "github.com/Sirupsen/logrus"

// scan without ClamAV returns empty result
func scan(path string) (string, error) {
	logrus.Debug("ClamAV is not configured to run")
	return "", nil
}
