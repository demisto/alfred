// +build clamav

package bot

// This file handles the ClamAV integration and is only built when specifying -tags clamav to build
// On OSX, you do the following to get all set up
// 1. Download the latest stable version of ClamAV and extract it to ~/demisto/clamav-0.98.7 (version might change)
// 2. cd ~/demisto/clamav-0.98.7
// 3. CFLAGS="-O3 -march=nocona" CXXFLAGS="-O3 -march=nocona" ./configure --build=x86_64-apple-darwin`uname -r` --enable-llvm
// 4. CFLAGS="-O3 -march=nocona" CXXFLAGS="-O3 -march=nocona" make
// 5. CFLAGS="-O3 -march=nocona" CXXFLAGS="-O3 -march=nocona" make check
// 6. CGO_CFLAGS=-I/Users/YOURUSERNAME/demisto/clamav-0.98.7/libclamav CGO_LDFLAGS=-L/Users/YOURUSERNAME/demisto/clamav-0.98.7/libclamav/.libs go get github.com/mirtchovski/clamav
// Notice in the above to replace to your username as well as the actual version of clamav
// Now, download the current database
// 7. curl http://database.clamav.net/main.cvd > main.cvd
// 8. curl http://database.clamav.net/daily.cvd > daily.cvd
// 9. curl http://database.clamav.net/bytecode.cvd > bytecode.cvd
// Now, you are ready to build alfred with the clamav tag and run it with the DB directory location pointing to ~/demisto/clamav-0.98.7
// DYLD_LIBRARY_PATH=/Users/YOURUSERNAME/demisto/clamav-0.98.7/libclamav/.libs ./alfred --loglevel=debug --clamdb=/Users/YOURUSERNAME/demisto/clamav-0.98.7

// On Ubuntu, it's all very simple. Just sudo apt-get install clamav livclamav6 libclamav-dev and no need for flags, etc.

import (
	"flag"
	"sync"

	"github.com/Sirupsen/logrus"
	"github.com/mirtchovski/clamav"
)

var (
	engine   *clamav.Engine
	clamdb   = flag.String("clamdb", clamav.DBDir(), "Directory where we can find the ClamAV definition database")
	initOnce sync.Once
	onceerr  error
)

func initClamAV() error {
	err := clamav.Init(clamav.InitDefault)
	if err != nil {
		return err
	}
	engine = clamav.New()
	sigs, err := engine.Load(*clamdb, clamav.DbStdopt)
	if err != nil {
		logrus.Errorf("Cannot initialize ClamAV engine: %v", err)
		return err
	}
	logrus.Debugf("Loaded %d signatures", sigs)
	engine.Compile()
	return nil
}

// scan the given bytes (file) using clamav and return the virus name
func scan(path string) (string, error) {
	initOnce.Do(func() {
		err := initClamAV()
		if err != nil {
			onceerr = err
		}
	})
	if onceerr != nil {
		return "", onceerr
	}
	virus, _, err := engine.ScanFile(path, clamav.ScanStdopt|clamav.ScanAllmatches)
	return virus, err
}
