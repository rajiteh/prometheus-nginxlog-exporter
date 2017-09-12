package main

import (
	"io/ioutil"
	"regexp"
	"sort"

	"github.com/hashicorp/hcl"
)

// StartupFlags is a struct containing options that can be passed via the
// command line
type StartupFlags struct {
	ConfigFile  string
	Filenames   []string
	Format      string
	Application string
	ListenPort  int
}

// Config models the application's configuration
type Config struct {
	Listen       ListenConfig
	Applications []ApplicationConfig `hcl:"application"`
}

// ListenConfig is a struct describing the built-in webserver configuration
type ListenConfig struct {
	Port    int
	Address string
}

// ApplicationConfig is a struct describing a single nginx application
type ApplicationConfig struct {
	Name     string            `hcl:",key"`
	LogFiles []string          `hcl:"log_files"`
	Format   string            `hcl:"format"`
	Labels   map[string]string `hcl:"labels"`
	Paths    []PathConfig      `hcl:"path"`

	OrderedLabelNames  []string
	OrderedLabelValues []string
}

// OrderLabels builds two lists of label keys and values, ordered by label name
func (c *ApplicationConfig) OrderLabels() {
	keys := make([]string, 0, len(c.Labels))
	values := make([]string, len(c.Labels))

	for k := range c.Labels {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	for i, k := range keys {
		values[i] = c.Labels[k]
	}

	c.OrderedLabelNames = keys
	c.OrderedLabelValues = values
}

type PathConfig struct {
	Pattern     string `hcl:",key"`
	ReplaceWith string `hcl:"replacewith"`
	Ignore      bool   `hcl:"ignore"`

	compiledPattern *regexp.Regexp
}

func (p *PathConfig) CompiledPattern() *regexp.Regexp {
	if p.compiledPattern == nil {
		p.compiledPattern = regexp.MustCompile(p.Pattern)
	}
	return p.compiledPattern
}

// LoadConfigFromFile fills a configuration object (passed as parameter) with
// values read from a configuration file (pass as parameter by filename). The
// configuration file needs to be in HCL format.
func LoadConfigFromFile(config *Config, filename string) error {
	buf, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	hclText := string(buf)

	err = hcl.Decode(config, hclText)
	if err != nil {
		return err
	}
	return nil
}

// LoadConfigFromFlags fills a configuration object (passed as parameter) with
// values from command-line flags.
func LoadConfigFromFlags(config *Config, flags *StartupFlags) error {
	config.Listen = ListenConfig{
		Port:    flags.ListenPort,
		Address: "0.0.0.0",
	}
	config.Applications = []ApplicationConfig{
		ApplicationConfig{
			Format:   flags.Format,
			LogFiles: flags.Filenames,
			Name:     flags.Application,
		},
	}
	return nil
}
