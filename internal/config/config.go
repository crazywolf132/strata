package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strata/internal/logs"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	GlobalConfigDirName = ".strata"
	GlobalConfigFile    = "global_config.yaml"
	LocalConfigFile     = "strata_repo.yaml"
)

func getXDGConfigPath() (string, error) {
	// XDG_CONFIG_HOME or fallback to `~/.config`
	xdg := os.Getenv("XDG_CONFIG_HOME")
	if xdg == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		xdg = filepath.Join(home, ".config")
	}
	p := filepath.Join(xdg, "strata", "config.yaml")
	return p, nil
}

var (
	globalConfig = make(map[string]string)
	localConfig  = make(map[string]string)

	globalLoaded bool
	localLoaded  bool
)

func InitializeGlobalConfig() error {
	if globalLoaded {
		return nil
	}

	configPath, err := getXDGConfigPath()
	if err != nil {
		return err
	}

	// ensure directory
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create XDG config dir: %v", err)
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// create minimal default
		def := map[string]string{"daemon_enabled": "false"}
		if e := saveYAML(configPath, def); e != nil {
			return e
		}
	}

	data, err := loadYAML(configPath)
	if err != nil {
		return err
	}
	for k, v := range data {
		globalConfig[k] = v
	}
	globalLoaded = true
	logs.Debug("Loaded global config from %s", configPath)
	return nil
}

func InitializeRepoConfig() error {
	if localLoaded {
		return nil
	}
	localPath := filepath.Join(".", LocalConfigFile)
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		def := map[string]string{
			"repo_name": guessRepoName(),
		}
		if err := saveYAML(localPath, def); err != nil {
			return err
		}
	}
	data, err := loadYAML(localPath)
	if err != nil {
		return err
	}
	for k, v := range data {
		localConfig[k] = v
	}
	localLoaded = true
	return nil
}

func GetConfigValue(key string) string {
	// local overrides global
	if val, ok := localConfig[key]; ok {
		return val
	}
	if val, ok := globalConfig[key]; ok {
		return val
	}
	return ""
}

func SetConfigValue(key, value string, global bool) error {
	if global {
		configPath, err := getXDGConfigPath()
		if err != nil {
			return err
		}
		globalConfig[key] = value
		return saveYAML(configPath, globalConfig)
	}
	// local
	localConfig[key] = value
	localPath := filepath.Join(".", LocalConfigFile)
	return saveYAML(localPath, localConfig)
}

func saveYAML(path string, data map[string]string) error {
	out, err := yaml.Marshal(data)
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0644)
}

func loadYAML(path string) (map[string]string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	d := make(map[string]string)
	if err := yaml.Unmarshal(content, &d); err != nil {
		return nil, err
	}
	return d, nil
}

func guessRepoName() string {
	cwd, _ := os.Getwd()
	parts := strings.Split(cwd, string(os.PathSeparator))
	return parts[len(parts)-1]
}
