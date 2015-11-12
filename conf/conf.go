// Package conf provides basic configuration handling from a file exposing a single global struct with all configuration.
package conf

import (
	"encoding/json"
	"io"
	"io/ioutil"

	"github.com/Sirupsen/logrus"
)

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
	// AWS credentials
	AWS struct {
		// ID to use
		ID string
		// Secret access key
		Secret           string
		ConfQueueName    string
		MessageQueueName string
		WorkQueueName    string
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
	Web     bool
	Bot     bool
	Dedup   bool
	Worker  bool
	ClamCtl string
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
	defOptions := []byte(`{
      "Env": "DEV",
			"Address": ":7070",
			"HTTPAddress": ":80",
			"ExternalAddress": "http://localhost:7070",
      "DB": {"ConnectString": "alfred.db"},
			"AWS": {"ConfQueueName": "TestConf", "MessageQueueName": "TestMessage", "WorkQueueName": "TestWork"},
			"G": {"ConfName": "conf", "MessageName": "msg", "WorkName": "work"},
			"Web": true,
			"Bot": true,
			"Dedup": true,
			"Worker": true,
			"ClamCtl": "/var/run/clamav/clamd.ctl"
    }`)
	// Start the options with the defaults and override with the file
	err := json.Unmarshal(defOptions, &Options)
	if err != nil {
		return err
	}
	options, err := ioutil.ReadFile(filename)
	if err != nil {
		if !useDefault {
			logrus.WithField("error", err).Warn("Could not open config file and not using default")
			return err
		}
		logrus.WithField("error", err).Info("Could not open config file - using defaults")
	} else {
		err = json.Unmarshal(options, &Options)
		if err != nil {
			return err
		}
	}
	finalOptions, err := json.MarshalIndent(&Options, "", "  ")
	if err != nil {
		return err
	}
	logrus.Infof("Using options:\n%s\n", string(finalOptions))
	return nil
}
