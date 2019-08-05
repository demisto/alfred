// Package conf provides basic configuration handling from a file exposing a single global struct with all configuration.
package conf

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"

	"github.com/Sirupsen/logrus"
)

var DefaultHelpMessage = `Here are the commands I understand when you send me a DIRECT MESSAGE here:
*config*: list the current channels I'm listening on
*join all/#channel1,#channel2...*: I will join all/specified public channels and start monitoring them.
*verbose on/off #channel1,#channel2,private1...* - turn on verbose mode on the specified channels or private groups
verbose mode is usually used by security professionals. When in verbose mode, dbot will display reputation details about any URL, IP or file including clean ones.

*af the-api-key-you-got-from-autofocus*: add your own AutoFocus credentials to use. Accepts "-" to return to default. 
*vt the-api-key-you-got-from-vt*: add your own VirusTotal key to use. Accepts "-" to return to default. You can get a key at https://www.virustotal.com/en/documentation/public-api/
*xfe the-api-key-you-got-from-xfe the-password-you-got*: add your own IBM X-Force Exchange credentials to use. Accepts "-" to return to default. You can get credentials at https://exchange.xforce.ibmcloud.com/
- It's important to specify your own keys to get reliable results as our public API keys are rate limited.`

// Options anonymous struct holds the global configuration options for the server
var Options struct {
	// The type of environment - PROD/TEST/DEV
	Env string
	// The address to listen on
	Address string
	// The HTTP address to listen on if the main address is HTTPS
	HTTPAddress string
	// ExternalAddress to our web tier
	ExternalAddress string
	// Security defintions
	Security struct {
		// The secret session key that is used to symmetrically encrypt sessions stored in cookies
		SessionKey string
		// Session timeout in minutes
		Timeout int
		// Recaptha secret
		Recaptcha string
		// Database encryption key used to encrypt the tokens
		DBKey string
	}
	// SSL configuration
	SSL struct {
		// The certificate file
		Cert string
		// The private key file
		Key string
	}
	// Slack application credentials
	Slack struct {
		// ClientID is passed to the OAuth request
		ClientID string
		// ClientSecret is used to verify Slack reply
		ClientSecret string
	}
	// VT token
	VT string
	// XFE credentials
	XFE struct {
		// Key to access the service
		Key string
		// Password to access the service
		Password string
	}
	// Cy API key
	Cy string
	// AF key
	AF string
	// DB properties
	DB struct {
		// ConnectString how to connect to DB
		ConnectString string
		// Username for the DB
		Username string
		// Password for DB
		Password string
		// ServerCA for TLS
		ServerCA string
		// ClientCert for TLS
		ClientCert string
		// ClientKey for TLS
		ClientKey string
	}
	G struct {
		Project     string
		ConfName    string
		MessageName string
		WorkName    string
		Credentials struct {
			PrivateKeyID string `json:"private_key_id"`
			PrivateKey   string `json:"private_key"`
			ClientEmail  string `json:"client_email"`
			ClientID     string `json:"client_id"`
			Type         string `json:"type"`
		}
	}
	Web       bool
	Worker    bool
	ClamCtl   string
	QueuePoll int
}

// The pipe writer to wrap around standard logger. It is configured in main.
var LogWriter *io.PipeWriter

// IsDev checks if we are running in the development environment.
func IsDev() bool {
	return Options.Env == "DEV"
}

// Load loads configuration from a file.
// If useDefault is provided then if there is an issue with the file we will use defaults.
func Load(filename string, useDefault bool) error {
	defOptions := []byte(`
{
	"Env": "DEV",
	"Address": ":7070",
	"HTTPAddress": ":80",
	"ExternalAddress": "http://localhost:7070",
	"DB": {
		"ConnectString": "alfred.db"
	},
	"Web": true,
	"Bot": true,
	"Worker": true,
	"ClamCtl": "/var/run/clamav/clamd.ctl",
	"QueuePoll": 10,
	"Security": {
		"SessionKey": "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
		"Timeout": 525600,
		"Recaptcha": "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx_xx_xxx",
		"DBKey": "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
	}
}`)
	// Start the options with the defaults and override with the file
	err := json.Unmarshal(defOptions, &Options)
	if err != nil {
		return err
	}
	if filename != "" {
		options, err := ioutil.ReadFile(filename)
		if err != nil {
			if !useDefault {
				logrus.WithError(err).Warn("Could not open config file and not using default")
				return err
			}
			logrus.WithError(err).Info("Could not open config file - using defaults")
		} else {
			err = json.Unmarshal(options, &Options)
			if err != nil {
				return err
			}
		}
	} else if !useDefault {
		logrus.Warn("no file provided and we are not using default")
		return errors.New("no file and no default")
	}
	finalOptions, err := json.MarshalIndent(&Options, "", "  ")
	if err != nil {
		return err
	}
	logrus.Infof("Using options:\n%s\n", string(finalOptions))
	return nil
}
