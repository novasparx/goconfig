package goconfig

import (
	"errors"
	"fmt"
	"io/ioutil"
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

//TODO: Have a Load from file with path (and no fatal or panic but returning error) and reuse it in LoadDefault instead of load()

func (c *Config) load() {
	if spaths, ok := getSecretsStoresPath(); ok {
		log.Println("Loading secrets")
		secp := Provider(spaths...)
		err := c.Koanf.Load(secp, nil)

		if err != nil {
			log.Fatal("Error loading secrets\n")
		}
	}

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

	log.Println("No config file found in default locations. Loading configuration from ENV variables and secrets only")
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

func getSecretsStoresPath() ([]string, bool) {
	defaultLoc := "/mnt/secrets-store"

	if !fileExists(defaultLoc) {
		return nil, false
	}

	sfis, err := ioutil.ReadDir(defaultLoc)

	if err != nil {
		log.Fatal("Could not get config secrets store directory\n")
	}

	var paths []string
	for _, d := range sfis {
		p, _ := filepath.Abs(defaultLoc + "/" + d.Name())

		if err != nil {
			log.Println("A secret store path could not be generated")
			continue
		}

		paths = append(paths, p)
	}

	return paths, true
}

type MountedVolumesProvider struct {
	paths []string
}

func Provider(paths ...string) *MountedVolumesProvider {
	ps := make([]string, 0)
	for _, p := range paths {
		ps = append(ps, filepath.Clean(p))
	}
	return &MountedVolumesProvider{paths: ps}
}

func (p *MountedVolumesProvider) ReadBytes() ([]byte, error) {
	return nil, errors.New("mounted volume provider does not support this method")
}

func (p *MountedVolumesProvider) Read() (map[string]interface{}, error) {
	conf := make(map[string]interface{})
	for _, p := range p.paths {
		fi, err := os.Stat(p)

		if err != nil {
			return nil, err
		}

		if !fi.IsDir() {
			b, err := ioutil.ReadFile(p)

			if err != nil {
				return nil, err
			}

			ks := strings.Split(fi.Name(), "-")
			unflatten(ks, string(b), conf)
			continue
		}

		fs, _ := ioutil.ReadDir(p)

		for _, f := range fs {
			if f.IsDir() {
				log.Println("More than one level of secrets store subdirectories not supported. Skipping.")
				continue
			}

			b, err := ioutil.ReadFile(p + "/" + f.Name())

			if err != nil {
				return nil, err
			}

			ks := strings.Split(f.Name(), "-")
			unflatten(ks, string(b), conf)
		}
	}

	return conf, nil
}

func unflatten(ks []string, v interface{}, m map[string]interface{}) {
	if len(ks) == 0 {
		return
	}

	if len(ks) == 1 {
		if _, ok := m[ks[0]]; !ok {
			m[ks[0]] = v
			return
		}
	}

	if sub, ok := m[ks[0]]; ok {
		unflatten(ks[1:], v, sub.(map[string]interface{}))
	} else {
		m[ks[0]] = make(map[string]interface{})
		unflatten(ks[1:], v, m[ks[0]].(map[string]interface{}))
	}
}
