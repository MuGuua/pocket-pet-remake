package config

import (
	"fmt"
	"os"
	"strings"
)

// LoadDefaultYAMLFile resolves the authoritative server config file path.
// The project now treats YAML as the single source of truth, while the
// optional `PP_CONFIG_FILE` environment variable only decides which YAML file
// to read instead of carrying the runtime values itself.
func LoadDefaultYAMLFile() (string, error) {
	paths := []string{}
	if explicit := strings.TrimSpace(os.Getenv("PP_CONFIG_FILE")); explicit != "" {
		paths = append(paths, explicit)
	} else {
		paths = append(paths,
			"backend/server/configs/config.yaml",
			"backend/server/configs/config.yml",
			"server/configs/config.yaml",
			"server/configs/config.yml",
			"configs/config.yaml",
			"configs/config.yml",
		)
	}

	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", err
		}
		if info.IsDir() {
			return "", fmt.Errorf("config path %s is a directory, want yaml file", path)
		}
		return path, nil
	}
	return "", fmt.Errorf("yaml config file not found; set PP_CONFIG_FILE or create backend/server/configs/config.yaml")
}
