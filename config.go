package main

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
