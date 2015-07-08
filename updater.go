package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"text/template"
)

type Application struct {
	Configs      []ServerConfig
	TemplateName string
	Template     *template.Template
	OutputFile   string
}

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
	ServerName        string `json:"server_name"`
	SSLCertificate    string `json:"ssl_certificate"`
	SSLCertificateKey string `json:"ssl_certificate_key"`

	UWSGILocations []UWSGILocation `json:"uwsgi_locations"`
	ProxyLocations []ProxyLocation `json:"proxy_locations"`
	Aliases        []Alias         `json:"aliases"`
	Rewrites       []Rewrite       `json:"rewrites"`
}

func warn_err(err error) {
	if err != nil {
		log.Println(err)
	}
}

func fail_err(err error) {
	if err != nil {
		panic(err)
	}
}

func (app *Application) reconfigureNginx() {
	f, err := os.Create(app.OutputFile)
	fail_err(err)
	defer f.Close()

	app.Template.Execute(f, app.Configs)
}

func (app *Application) updateConfig(w http.ResponseWriter, r *http.Request) {
	var err error
	var body []byte
	var config ServerConfig

	defer r.Body.Close()
	body, err = ioutil.ReadAll(r.Body)
	warn_err(err)

	err = json.Unmarshal(body, &config)
	warn_err(err)

	app.Configs = append(app.Configs, config)

	go app.reconfigureNginx()
}

func main() {
	var err error

	app := Application{}

	flag.StringVar(&app.TemplateName, "template", "default.conf.tmpl", "Config file template to be rendered. Default: default.conf.tmpl")
	flag.StringVar(&app.OutputFile, "out", "/etc/nginx/conf.d/default.conf", "Path to the config file to be updated. Default: /etc/nginx/conf.d/default.conf")
	flag.Parse()

	app.Template, err = template.ParseFiles(app.TemplateName)
	fail_err(err)

	http.HandleFunc("/updateConfig", app.updateConfig)
	log.Fatal(http.ListenAndServe(":3456", nil))
}
