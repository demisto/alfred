#!/bin/bash
#
# Simple script to set up Circle environment and run the tests

BUILD_DIR=$HOME/go
GO_VERSION=go1.5
TIMEOUT="-timeout 30s"

# Executes the given statement, and exits if the command returns a non-zero code.
function exit_if_fail {
  command=$@
  echo "Executing '$command'"
  $command
  rc=$?
  if [ $rc -ne 0 ]; then
    echo "'$command' returned $rc."
    exit $rc
  fi
}

# Check that go fmt has been run.
function check_go_fmt {
  fmtcount=`git ls-files | grep '.go$' | xargs gofmt -l 2>&1 | wc -l`
  if [ $fmtcount -gt 0 ]; then
    echo "run 'go fmt ./...' to format your source code."
    exit 1
  fi
}

# Check that go vet passes.
function check_go_vet {
  vetcount=`go vet ./... 2>&1 | wc -l`
  if [ $vetcount -gt 0 ]; then
    echo "run 'go vet ./...' to see the errors it flags and correct your source code."
    exit 1
  fi
}

source $HOME/.gvm/scripts/gvm
exit_if_fail gvm use $GO_VERSION

# Set up the build directory, and then GOPATH.
exit_if_fail mkdir $BUILD_DIR
export GOPATH=$BUILD_DIR
exit_if_fail mkdir -p $GOPATH/src/github.com/demisto

# Dump some test config to the log.
echo "Configuration"
echo "========================================"
echo "\$HOME: $HOME"
echo "\$GOPATH: $GOPATH"
echo "\$GO_VERSION: $GO_VERSION"

# Move the checked-out source to a better location.
exit_if_fail mv $HOME/alfred $GOPATH/src/github.com/demisto

# Install the code.
exit_if_fail cd $GOPATH/src/github.com/demisto/alfred
exit_if_fail go get -t -d -v ./...

# Touch missing private files
exit_if_fail touch $GOPATH/src/github.com/demisto/alfred/static/master/jade/_gtmid.json
echo '{}' >> $GOPATH/src/github.com/demisto/alfred/static/master/jade/_gtmid.json
exit_if_fail touch $GOPATH/src/github.com/demisto/alfred/static/master/jade/_ze.jade
exit_if_fail touch $GOPATH/src/github.com/demisto/alfred/static/master/jade/_analytics.jade

# Build all the web stuff
exit_if_fail cd $GOPATH/src/github.com/demisto/alfred/static/master
exit_if_fail npm install
exit_if_fail bower install
exit_if_fail gulp build

# Embed the static files inside the executable
exit_if_fail go get github.com/slavikm/esc
exit_if_fail cd $GOPATH/src/github.com/demisto/alfred
exit_if_fail $GOPATH/bin/esc -o web/static.go -pkg web -prefix static/site/ static/site/
check_go_fmt
check_go_vet
exit_if_fail go build -v ./...

# Finally, test

go test $TIMEOUT -v ./...
exit $?
