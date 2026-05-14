package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func LoadEnvFileIfPresent(path string) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return false, fmt.Errorf("parse %s:%d: missing '='", path, lineNo)
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" {
			return false, fmt.Errorf("parse %s:%d: empty key", path, lineNo)
		}
		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}

		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		if err := os.Setenv(key, value); err != nil {
			return false, fmt.Errorf("set env %s from %s:%d: %w", key, path, lineNo, err)
		}
	}
	if err := scanner.Err(); err != nil {
		return false, err
	}
	return true, nil
}

func LoadDefaultEnvFiles() (string, error) {
	paths := []string{}
	if explicit := strings.TrimSpace(os.Getenv("PP_CONFIG_FILE")); explicit != "" {
		paths = append(paths, explicit)
	} else {
		paths = append(paths,
			"backend/server/configs/config.env",
			"server/configs/config.env",
			"configs/config.env",
		)
	}

	for _, path := range paths {
		loaded, err := LoadEnvFileIfPresent(path)
		if err != nil {
			return "", err
		}
		if loaded {
			return path, nil
		}
	}
	return "", nil
}
