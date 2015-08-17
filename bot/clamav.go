// +build clamav

package bot

// This file handles the ClamAV integration and is only built when specifying -tags clamav to build
// On OSX, you do the following to get all set up - you need of course Xcode and the command line tools
// See http://www.gctv.ne.jp/~yokota/clamav/
// 1. Download the latest stable version of ClamAV and extract it to ~/demisto/clamav-0.98.7 (version might change)
// 2. cd ~/demisto/clamav-0.98.7
// 3. CFLAGS="-O3 -march=nocona" CXXFLAGS="-O3 -march=nocona" ./configure --build=x86_64-apple-darwin`uname -r` --enable-llvm=no
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

// On Ubuntu, it's all very simple. Just sudo apt-get install clamav libclamav6 libclamav-dev and no need for flags, etc.

import (
	"bytes"
	"flag"
	"io"
	"net"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/demisto/alfred/conf"
	"github.com/mirtchovski/clamav"
)

var (
	clamdb = flag.String("clamdb", clamav.DBDir(), "Directory where we can find the ClamAV definition database")
)

type clamEngine struct {
	engine *clamav.Engine
	l      net.Listener
}

func newClamEngine() (*clamEngine, error) {
	err := clamav.Init(clamav.InitDefault)
	if err != nil {
		return nil, err
	}
	l, err := net.Listen("unix", conf.Options.ClamCtl)
	if err != nil {
		return nil, err
	}
	ce := &clamEngine{engine: clamav.New()}
	err = ce.loadSigs()
	if err != nil {
		return nil, err
	}
	ce.l = l
	go ce.listenUpdate()
	return ce, nil
}

func (ce *clamEngine) loadSigs() error {
	sigs, err := ce.engine.Load(*clamdb, clamav.DbStdopt)
	if err != nil {
		logrus.Errorf("Cannot initialize ClamAV engine: %v", err)
		return err
	}
	logrus.Debugf("Loaded %d signatures", sigs)
	ce.engine.Compile()
	return nil
}

func (ce *clamEngine) listenUpdate() {
	for {
		c, err := ce.l.Accept()
		if err != nil {
			logrus.Debugln("Shutting down ClamAV engine update")
			return
		}
		b := &bytes.Buffer{}
		_, err = io.Copy(b, c)
		if err != nil {
			logrus.Errorf("Error updating from freshclam - %v\n", err)
			continue
		}
		reload := b.String()
		if reload != "RELOAD" {
			logrus.Infof("Weird - got %s from freshclam\n", reload)
		}
		_, err = c.Write([]byte("RELOADING"))
		if err != nil {
			logrus.Infof("Error updating freshclam - %v\n", err)
		}
	}
}

func (ce *clamEngine) close() {
	ce.l.Close()
	os.Remove(conf.Options.ClamCtl)
	ce.engine.Free()
}

// scan the given bytes (file) using clamav and return the virus name
func (ce *clamEngine) scan(filename string, b []byte) (string, error) {
	fmap := clamav.OpenMemory(b)
	defer clamav.CloseMemory(fmap)

	virus, _, err := ce.engine.ScanMapCb(fmap, clamav.ScanStdopt|clamav.ScanBlockbroken, filename)
	return virus, err
}
