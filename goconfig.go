package goconfig

import (
	"io/fs"
	"log"
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

	envp := env.Provider("", ".", func(s string) string {
		return strings.Replace(strings.ToLower(s), "_", ".", -1)
	})

	pn := filepath.Base(os.Args[0])
	cn := pn + ".yml"

	switch pn {
	case "main":
		cn = "config.yml"
	case "__debug_bin":
		//dlv debug session
		_, fn, _, ok := runtime.Caller(1)

		if !ok {
			log.Fatalf("error loading config")
		}

		fnb := filepath.Base(fn)

		if fnb == "main.go" {
			cn = "config.yml"
			break
		}

		s := strings.TrimSuffix(fnb, filepath.Ext(fn))
		cn = s + ".yml"
	}

	wd := filepath.Dir(os.Args[0])
	err := k.Load(file.Provider(wd+"/config/"+cn), yaml.Parser())

	if err == nil {
		log.Printf("Config file found in %s", wd+"/config/"+cn)
		k.Load(envp, nil)
		return &Config{k}
	}

	switch err.(type) {
	case *fs.PathError:
	default:
		log.Fatalf("error loading config: %v", err)
	}

	_, fn, _, ok := runtime.Caller(0)

	if !ok {
		log.Fatalf("error loading config: %v", err)
	}

	wd = filepath.Dir(fn)

	err = k.Load(file.Provider(wd+"/config/"+cn), yaml.Parser())

	if err == nil {
		log.Printf("Config file found in %s", wd+"/config/"+cn)
		k.Load(envp, nil)
		return &Config{k}
	}

	switch err.(type) {
	case *fs.PathError:
	default:
		log.Fatalf("error loading config: %v", err)
	}

	wd, err = os.Getwd()

	if !ok {
		log.Fatalf("error loading config: %v", err)
	}

	err = k.Load(file.Provider(wd+"/config/"+cn), yaml.Parser())

	if err == nil {
		log.Printf("Config file found in %s", wd+"/config/"+cn)
		k.Load(envp, nil)
		return &Config{k}
	}

	switch err.(type) {
	case *fs.PathError:
		log.Println("No config file found. Loading configuration from ENV variables only")
	default:
		log.Fatalf("error loading config: %v", err)
	}

	err = k.Load(envp, nil)

	if err != nil {
		log.Fatalf("error loading config: %v", err)
	}

	return &Config{k}
}
