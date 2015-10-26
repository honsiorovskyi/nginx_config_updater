package main

import (
	"encoding/json"
	"os"
)

type UWSGILocation struct {
	Path      string `json:"path"`
	UWSGIAddr string `json:"uwsgi_addr"`
	UWSGIPort string `json:"uwsgi_port"`
}

type ProxyLocation struct {
	Path     string `json:"path"`
	ProxyURL string `json:"proxy_url"`
}

type Alias struct {
	Path      string `json:"path"`
	LocalPath string `json:"local_path"`
}

type Rewrite struct {
	Path string `json:"path"`
	Rule string `json:"rule"`
}

type ServerConfig struct {
	Id                string `json:"id"`
	ServerName        string `json:"server_name"`
	SSLCertificate    string `json:"ssl_certificate"`
	SSLCertificateKey string `json:"ssl_certificate_key"`

	UWSGILocations []UWSGILocation `json:"uwsgi_locations"`
	ProxyLocations []ProxyLocation `json:"proxy_locations"`
	Aliases        []Alias         `json:"aliases"`
	Rewrites       []Rewrite       `json:"rewrites"`
}

type UpstreamConfig struct {
	Id      string   `json:"id"`
	Servers []string `json:"servers"`
}

type NginxConfig struct {
	ServerConfigs   map[string]*ServerConfig   `json:"server_configs"`
	UpstreamConfigs map[string]*UpstreamConfig `json:"upstream_config"`
}

func NewNginxConfig() *NginxConfig {
	nc := new(NginxConfig)
	nc.ServerConfigs = make(map[string]*ServerConfig)
	nc.UpstreamConfigs = make(map[string]*UpstreamConfig)
	return nc
}

func (nc *NginxConfig) Load(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	if err := json.NewDecoder(file).Decode(nc); err != nil {
		return err
	}

	return nil
}

func (nc *NginxConfig) Save(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	if err := json.NewEncoder(file).Encode(nc); err != nil {
		return err
	}

	return nil
}
