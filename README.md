# DBOT - Demisto Security Bot [![Circle CI](https://circleci.com/gh/demisto/alfred/tree/master.svg?style=svg&circle-token=298d2e89802eaed2e8972abe83baac50d9ee5224)](https://circleci.com/gh/demisto/alfred/tree/master)

A Slack bot to add security info to messages containing URLs, hashes and IPs. You can see it in action at [dbot.demisto.com](https://dbot.demisto.com).

## Authors
This project was built by the [Demisto](https://www.demisto.com) team

## Quick Start

Make sure you have a Go environment set up (either using [GVM](https://github.com/moovweb/gvm/) or just native install)

```sh
$ go get -t -u -d -v github.com/demisto/alfred
```

To get the client artifacts (html, css, js) built install node, npm and then:
```sh
$ cd $GOPATH/src/github.com/demisto/alfred/client
$ npm i
$ npm run build
```
(this will create client artifacts under `$GOPATH/src/github.com/demisto/alfred/client/build`)

Create the Go wrapper around the client files:

```sh
$ go get -v github.com/slavikm/esc
$ cd $GOPATH/src/github.com/demisto/alfred/
$ $GOPATH/bin/esc -o web/static.go -pkg web -prefix client/build -ignore \\.DS_Store client/build
```

And finally, install and run:

```sh
$ cd $GOPATH/src/github.com/demisto/alfred/
$ go install
$ cd $GOPATH/bin
$ ./alfred [-loglevel debug] [-conf path/to/conf] [-logfile path/to/log]
```

If you are running from bin (as above), make sure to create a soft link to the site
```sh
$ ln -s ln -s $GOPATH/src/github.com/demisto/alfred/static/ static
```

Install `mysql`
Run the following to configure sql database:
```sh
$ mysql -u root (if password is set then add -p)
mysql> CREATE DATABASE demisto CHARACTER SET = utf8;
mysql> CREATE DATABASE demistot CHARACTER SET = utf8;
mysql> CREATE USER demisto IDENTIFIED BY 'password';
mysql> GRANT ALL on demisto.* TO demisto;
mysql> GRANT ALL on demistot.* TO demisto;
mysql> drop user ''@'localhost';
```


Or, you can run directly from the source without installing by:
```sh
$ cd $GOPATH/src/github.com/demisto/alfred/
$ go run alfred.go [-loglevel debug] [-conf path/to/conf] [-logfile path/to/log]
```

Please make sure to run esc again to embed the fully updated site into Go before release.
While developing, you don't need to run esc unless adding new files to the site.

### Configuration
- Make sure to specify the Slack client ID and secret in a configuration file
- To get VirusTotal reputation, you must specify the VirusTotal key. See conf/conf.go for more details.
- Configure mysql database configuration under `"DB"` key (See conf/conf.go for more detail):
```
{
    "ConnectString": "tcp(127.0.0.1:3306)/demisto?parseTime=true", // where "demisto" is the DATABASE name from previous step and 127.0.0.1:3306 is the ip:port of the database
    "Username": "demisto", // user created in previous step
    "Password": "password", // user's password created in previous step
    "ServerCA": "-----BEGIN CERTIFICATE---...", // Not necessary for local mysql
    "ClientCert": "-----BEGIN CERTIFICATE----...", // Not necessary for local mysql
    "ClientKey": "-----BEGIN RSA PRIVATE KEY--..." // Not necessary for local mysql
}
```

