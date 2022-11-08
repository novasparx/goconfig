package goconf

import (
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
)

type Config struct {
	*koanf.Koanf
}

func New() *Config {
	k := koanf.New(".")
	return &Config{k}
}

func LoadDefault() *Config {
	c := New()
	c.load()
	return c
}

//TODO: Have a Load from file with path and reuse it in LoadDefault instead of load()

func (c *Config) load() {
	envp := env.Provider("", ".", func(s string) string {
		return strings.Replace(strings.ToLower(s), "_", ".", -1)
	})

	if path, ok := findConfigPath(); ok {
		log.Printf("Config file found in %s", path)
		err := c.Koanf.Load(file.Provider(path), yaml.Parser())

		if err != nil {
			log.Fatalf("Error loading config: %v", err)
		}

		err = c.Koanf.Load(envp, nil)

		if err != nil {
			log.Fatalf("Error loading config: %v", err)
		}

		return
	}

	log.Println("No config file found. Loading configuration from ENV variables only")
	err := c.Koanf.Load(envp, nil)

	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}
}

func findConfigPath() (string, bool) {
	execp, err := filepath.Abs(os.Args[0])

	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	path := execp + ".yml"
	if fileExists(path) {
		return path, true
	}

	path = execp + ".yaml"
	if fileExists(path) {
		return path, true
	}

	path = filepath.Dir(execp) + "/config.yml"
	if fileExists(path) {
		return path, true
	}

	path = filepath.Dir(execp) + "/config.yaml"
	if fileExists(path) {
		return path, true
	}

	mfpath, err := findMainExecPath()

	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	mnoext := strings.TrimSuffix(mfpath, filepath.Ext(mfpath))

	path = mnoext + ".yml"
	if fileExists(path) {
		return path, true
	}

	path = mnoext + ".yaml"
	if fileExists(path) {
		return path, true
	}

	path = filepath.Dir(mnoext) + "/config.yml"
	if fileExists(path) {
		return path, true
	}

	path = filepath.Dir(mnoext) + "/config.yaml"
	if fileExists(path) {
		return path, true
	}

	base := filepath.Base(mnoext)
	wd, err := os.Getwd()

	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	path = wd + "/" + base + ".yml"
	if fileExists(path) {
		return path, true
	}

	path = wd + "/" + base + ".yaml"
	if fileExists(path) {
		return path, true
	}

	path = wd + "/config.yml"
	if fileExists(path) {
		return path, true
	}

	path = wd + "/config.yaml"
	if fileExists(path) {
		return path, true
	}

	return "", false
}

func fileExists(fn string) bool {
	if _, err := os.Stat(fn); err == nil {
		return true
	} else if errors.Is(err, os.ErrNotExist) {
		return false
	} else {
		log.Printf("Error checking file %s: %v", fn, err)
		return false
	}
}

func findMainExecPath() (string, error) {
	var i int
	for i = 0; i < math.MaxInt; i++ {
		_, _, _, ok := runtime.Caller(i)

		if !ok {
			break
		}
	}

	pcs := make([]uintptr, i)
	runtime.Callers(1, pcs)

	cfs := runtime.CallersFrames(pcs)

	c := true
	var f runtime.Frame
	for c {
		f, c = cfs.Next()

		if f.Function == "main.main" {
			return f.File, nil
		}
	}

	return "", fmt.Errorf("Could not find main.main function execution directory to locate config file")
}
