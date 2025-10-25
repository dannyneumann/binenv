package app

import (
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// GetDefaultBinDir returns the bin directory in usermode
func GetDefaultBinDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	d := filepath.Join(home, ".binenv") + string(os.PathSeparator)

	return d, nil
}

// GetDefaultLinkDir returns the bin directory in usermode
func GetDefaultLinkDir() (string, error) {
	return GetDefaultBinDir()
}

// GetDefaultConfDir returns the config directory in usermode
func GetDefaultConfDir() (string, error) {
	var err error

	d := os.Getenv("XDG_CONFIG_HOME")
	if d == "" {
		d, err = os.UserHomeDir()
		if err != nil {
			return "", err
		}
		d = filepath.Join(d, ".config")
	}

	return filepath.Join(d, "binenv"), nil
}

// GetDefaultCacheDir returns the cache directory in usermode
func GetDefaultCacheDir() (string, error) {
	var err error

	d := os.Getenv("XDG_CACHE_HOME")
	if d == "" {
		d, err = os.UserHomeDir()
		if err != nil {
			return "", err
		}
		d = filepath.Join(d, ".cache")
	}

	return filepath.Join(d, "binenv"), nil
}

func stringInSlice(st string, sl []string) bool {
	for _, v := range sl {
		if v == st {
			return true
		}
	}

	return false
}

func stringToEnvVarName(st string) string {
	reg, err := regexp.Compile("[^a-zA-Z0-9_]+")
	if err != nil {
		log.Fatal(err)
	}
	result := reg.ReplaceAllString(st, "_")
	return strings.ToUpper(result)
}
