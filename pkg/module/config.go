package module

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Package      PackageInfo           `toml:"package"`
	Dependencies map[string]Dependency `toml:"dependencies"`
	DevDeps      map[string]Dependency `toml:"dev-dependencies"`
	Build        BuildConfig           `toml:"build"`
	Scripts      map[string]string     `toml:"scripts"`
}

type PackageInfo struct {
	Name        string   `toml:"name"`
	Version     string   `toml:"version"`
	Description string   `toml:"description"`
	Authors     []string `toml:"authors"`
	License     string   `toml:"license"`
	Repository  string   `toml:"repository"`
	Homepage    string   `toml:"homepage"`
	Keywords    []string `toml:"keywords"`
	Entry       string   `toml:"entry"`
}

type Dependency struct {
	Version string `toml:"version"`
	Git     string `toml:"git"`
	Branch  string `toml:"branch"`
	Tag     string `toml:"tag"`
	Path    string `toml:"path"`
}

type BuildConfig struct {
	Output    string   `toml:"output"`
	Target    string   `toml:"target"`
	Optimize  bool     `toml:"optimize"`
	Debug     bool     `toml:"debug"`
	Includes  []string `toml:"includes"`
	Libraries []string `toml:"libraries"`
}

func LoadConfig(dir string) (*Config, error) {
	configPath := filepath.Join(dir, "carv.toml")

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) Save(dir string) error {
	configPath := filepath.Join(dir, "carv.toml")

	f, err := os.Create(configPath)
	if err != nil {
		return err
	}
	defer f.Close()

	encoder := toml.NewEncoder(f)
	return encoder.Encode(c)
}

func FindProjectRoot(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}

	for {
		configPath := filepath.Join(dir, "carv.toml")
		if _, err := os.Stat(configPath); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return dir, nil
		}
		dir = parent
	}
}

func DefaultConfig(name string) *Config {
	return &Config{
		Package: PackageInfo{
			Name:    name,
			Version: "0.1.0",
			Entry:   "src/main.carv",
		},
		Dependencies: make(map[string]Dependency),
		DevDeps:      make(map[string]Dependency),
		Build: BuildConfig{
			Output:   "build",
			Optimize: true,
		},
		Scripts: map[string]string{
			"build": "carv build src/main.carv",
			"run":   "carv run src/main.carv",
			"test":  "carv test",
		},
	}
}
