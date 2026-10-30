package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	_CONFIG_FILE = "config.yaml"
	_CONFIG_PERM = 0600
	_DIR_PERM    = 0700
)

var Current Config

func InitialLoad() error {
	cfg, err := load()
	if err == nil {
		Current = cfg
		return chmodConfig()
	}

	Current = defaultConfig
	if os.IsNotExist(err) {
		configDir, derr := getDir()
		if derr != nil {
			return derr
		}
		if oldToken, rerr := os.ReadFile(filepath.Join(configDir, "token")); rerr == nil {
			Current.Token = strings.TrimSpace(string(oldToken))
		}
		return save(Current)
	}

	if raw, rerr := os.ReadFile(Path()); rerr == nil {
		if tok := tokenFromYAML(raw); tok != "" {
			Current.Token = tok
		}
	}
	_ = chmodConfig()
	return err
}

func tokenFromYAML(content []byte) string {
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "token:") {
			continue
		}
		return strings.Trim(strings.TrimSpace(strings.TrimPrefix(line, "token:")), `"'`)
	}
	return ""
}

func chmodConfig() error {
	path := Path()
	if path == "" {
		return nil
	}
	if _, err := os.Stat(path); err != nil {
		return nil
	}
	return os.Chmod(path, _CONFIG_PERM)
}

func getDir() (string, error) {
	if testDir != "" {
		if err := os.MkdirAll(testDir, _DIR_PERM); err != nil {
			return "", err
		}
		if err := os.Chmod(testDir, _DIR_PERM); err != nil {
			return "", err
		}
		return testDir, nil
	}

	userDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	configDir := filepath.Join(userDir, ".config", DirName)
	err = os.MkdirAll(configDir, _DIR_PERM)
	if err != nil {
		return "", err
	}
	if err := os.Chmod(configDir, _DIR_PERM); err != nil {
		return "", err
	}

	return configDir, nil
}

func load() (Config, error) {
	configDir, err := getDir()
	if err != nil {
		return defaultConfig, err
	}

	configContent, err := os.ReadFile(filepath.Join(configDir, _CONFIG_FILE))
	if err != nil {
		return defaultConfig, err
	}

	var newConfig Config
	err = yaml.Unmarshal(configContent, &newConfig)
	if err != nil {
		return defaultConfig, err
	}

	if newConfig.VolumeStep == 0 {
		newConfig.VolumeStep = defaultConfig.VolumeStep
	}

	if newConfig.Search == nil {
		search := *defaultConfig.Search
		newConfig.Search = &search
	}

	if newConfig.Controls == nil {
		controls := *defaultConfig.Controls
		newConfig.Controls = &controls
	} else {
		fillDefault(newConfig.Controls, defaultConfig.Controls)
		if newConfig.Controls.Quit.IsEmpty() {
			newConfig.Controls.Quit = defaultConfig.Controls.Quit
		}
	}

	if newConfig.Style == nil {
		style := *defaultConfig.Style
		newConfig.Style = &style
	} else {
		if newConfig.Style.Icons == nil {
			icons := *defaultConfig.Style.Icons
			newConfig.Style.Icons = &icons
		} else {
			fillDefault(newConfig.Style.Icons, defaultConfig.Style.Icons)
		}
		if newConfig.Style.Colors == nil {
			colors := *defaultConfig.Style.Colors
			newConfig.Style.Colors = &colors
		} else {
			fillDefault(newConfig.Style.Colors, defaultConfig.Style.Colors)
		}
		if newConfig.Style.SidePanelWidth == 0 {
			newConfig.Style.SidePanelWidth = defaultConfig.Style.SidePanelWidth
		}
	}

	return newConfig, nil
}

func fillDefault(target, values any) {
	targetStruct := reflect.ValueOf(target).Elem()
	defaultStruct := reflect.ValueOf(values).Elem()
	for i := 0; i < targetStruct.NumField(); i++ {
		field := targetStruct.Field(i)
		if field.IsZero() || field.String() == "" {
			field.Set(defaultStruct.Field(i))
		}
	}
}

func save(conf Config) error {
	configDir, err := getDir()
	if err != nil {
		return err
	}

	path := filepath.Join(configDir, _CONFIG_FILE)
	if conf.Token == "" {
		if existing, rerr := os.ReadFile(path); rerr == nil {
			if tok := tokenFromYAML(existing); tok != "" {
				conf.Token = tok
			}
		}
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, _CONFIG_PERM)
	if err != nil {
		return err
	}
	defer file.Close()

	if err := file.Chmod(_CONFIG_PERM); err != nil {
		return err
	}

	enc := yaml.NewEncoder(file)
	enc.SetIndent(4)
	err = enc.Encode(conf)
	if err != nil {
		return err
	}

	return nil
}

func Path() string {
	configDir, err := getDir()
	if err != nil {
		return ""
	}

	return filepath.Join(configDir, _CONFIG_FILE)
}

func Save() error {
	return save(Current)
}

func Reset() error {
	var err error
	Current, err = load()
	return err
}
