package main

import (
	"io/ioutil"

	"github.com/miekg/dns"
	"gopkg.in/yaml.v3"
)

type Config struct {
	DSN     string                      `yaml:"dsn"`
	Forward map[string]ServerDefinition `yaml:"forward zones"`
	Reverse map[string]ServerDefinition `yaml:"reverse zones"`
}

type ServerDefinition struct {
	Server  string `yaml:"server"`
	Algo    string `yaml:"algo"`
	KeyName string `yaml:"keyname"`
	Secret  string `yaml:"secret"`
}

func (s *Config) Load(configfile string) *Config {
	data, err := ioutil.ReadFile(configfile)
	if err != nil {
		panic(err)
	}
	return s.parse(data)
}

func (s *Config) parse(configdata []byte) *Config {
	err := yaml.Unmarshal(configdata, &s)
	if err != nil {
		panic(err)
		// return nil, err
	}

	// Sanitization fwd and reverse
	for _, zonemap := range []map[string]ServerDefinition{s.Forward, s.Reverse} {
		for zonename, serverdef := range zonemap {
			if !dns.IsFqdn(serverdef.KeyName) {
				serverdef.KeyName = dns.Fqdn(serverdef.KeyName)
				zonemap[zonename] = serverdef
			}

			if !dns.IsFqdn(zonename) {
				zonemap[dns.Fqdn(zonename)] = zonemap[zonename]
				delete(zonemap, zonename)
			}
		}
	}

	return s
}
