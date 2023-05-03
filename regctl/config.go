package regctl

import "github.com/regclient/regclient/config"

// Config struct contains contents loaded from / saved to a config file
type Config struct {
	Filename      string                  `json:"-"`                 // filename that was loaded
	Version       int                     `json:"version,omitempty"` // version the file in case the config file syntax changes in the future
	Hosts         map[string]*config.Host `json:"hosts"`
	BlobLimit     int64                   `json:"blobLimit,omitempty"`
	IncDockerCert *bool                   `json:"incDockerCert,omitempty"`
	IncDockerCred *bool                   `json:"incDockerCred,omitempty"`
}
