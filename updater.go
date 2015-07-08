package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"net/http"
	"os"
	"text/template"
)

type Application struct {
	Configs      map[string]ServerConfig
	TemplateName string
	Template     *template.Template
	OutputFile   string
}

func (app *Application) Setup() {
	var err error

	app.Template, err = template.ParseFiles(app.TemplateName)
	fail_on_err(err)

	app.Configs = make(map[string]ServerConfig)
}

func (app *Application) reconfigureNginx() error {
	f, err := os.Create(app.OutputFile)
	if err != nil {
		return err
	}
	defer f.Close()

	app.Template.Execute(f, app.Configs)
	return nil
}

func (app *Application) DeleteConfig(w http.ResponseWriter, r *http.Request) {
	serverId := r.FormValue("id")

	if serverId == "" {
		http_err(w, "Server identifier is not specified", nil)
		return
	}

	delete(app.Configs, serverId)
	if err := app.reconfigureNginx(); err != nil {
		http_err(w, "Could not reconfigure nginx", err)
		return
	}
}

func (app *Application) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	var err error
	var body []byte
	var config ServerConfig

	defer r.Body.Close()
	body, err = ioutil.ReadAll(r.Body)
	if err != nil {
		http_err(w, "Request I/O error", err)
		return
	}

	err = json.Unmarshal(body, &config)
	if err != nil {
		http_err(w, "Data parsing error", err)
		return
	}

	if config.Id == "" {
		http_err(w, "Server identifier is not specified", err)
		return
	}

	app.Configs[config.Id] = config
	if err := app.reconfigureNginx(); err != nil {
		http_err(w, "Could not reconfigure nginx", err)
		return
	}
}

func main() {
	app := Application{}

	flag.StringVar(&app.TemplateName, "template", "default.conf.tmpl", "Config file template to be rendered. Default: default.conf.tmpl")
	flag.StringVar(&app.OutputFile, "out", "/etc/nginx/conf.d/default.conf", "Path to the config file to be updated. Default: /etc/nginx/conf.d/default.conf")
	flag.Parse()

	app.Setup()

	http.HandleFunc("/updateConfig", app.UpdateConfig)
	http.HandleFunc("/deleteConfig", app.DeleteConfig)

	fail_on_err(http.ListenAndServe(":3456", nil))
}
