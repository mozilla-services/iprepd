package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
)

const defaultConfigFilename = ".repd"

// Config is the iprepd cli configuration
type Config struct {
	HostURL string `json:"host_url"`
	AuthTK  string `json:"auth_token"`
}

// GetDefaultPath returns the best place to save / look-for a config file.
// Using this function for both saving and reading the config file (with the
// same OS) guarantees that a file will be found
func GetDefaultPath() string {
	usr, err := user.Current()
	if err != nil {
		// settle for rootdir
		return fmt.Sprintf("/%s", defaultConfigFilename)
	}
	return fmt.Sprintf("%s/%s", usr.HomeDir, defaultConfigFilename)
}

// SetConfig writes a configuration file to the given path
func SetConfig(url, tk, path string) error {
	if url == "" {
		return errors.New("url cannot be empty")
	}
	if tk == "" {
		return errors.New("token cannot be empty")
	}
	if path == "" {
		path = GetDefaultPath()
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("could not create new file %s: %s", path, err)
	}
	byt, err := json.Marshal(&Config{HostURL: url, AuthTK: tk})
	if err != nil {
		return fmt.Errorf("could not marshal configuration file: %s", err)
	}
	if _, err := f.Write(byt); err != nil {
		return fmt.Errorf("could not write configuration file: %s", err)
	}
	return nil
}

// GetConfig returns the configuration at a given path
func GetConfig(path string) (*Config, error) {
	if path == "" {
		path = GetDefaultPath()
	}
	return readFSConfig(path)
}

func readFSConfig(path string) (*Config, error) {
	dat, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read configuration file %s: %s", path, err)
	}
	var c *Config
	if err = json.Unmarshal(dat, &c); err != nil {
		return nil, fmt.Errorf("could not unmarshal config: %s", err)
	}
	return c, nil
}
