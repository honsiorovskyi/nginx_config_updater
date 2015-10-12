package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"text/template"
)

type Application struct {
	Configs      map[string]*ServerConfig
	TemplateName string
	Template     *template.Template
	OutputFile   string
}

func (app *Application) Setup() {
	var err error

	app.Template, err = template.ParseFiles(app.TemplateName)
	fail_on_err(err)

	app.Configs = make(map[string]*ServerConfig)
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

	// delete config from map
	delete(app.Configs, serverId)

	// reconfigure nginx
	if err := app.reconfigureNginx(); err != nil {
		http_err(w, "Could not reconfigure nginx", err)
		return
	}
}

func (app *Application) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	var config ServerConfig

	defer r.Body.Close()
	err := json.NewDecoder(r.Body).Decode(&config)
	if err != nil {
		http_err(w, "Data parsing error", err)
		return
	}

	// validate config id
	if config.Id == "" {
		http_err(w, "Server identifier is not specified", err)
		return
	}

	// add/update config to map
	app.Configs[config.Id] = &config

	// reconfigure nginx
	if err := app.reconfigureNginx(); err != nil {
		http_err(w, "Could not reconfigure nginx", err)
		return
	}
}

func (app *Application) AddUpstream(w http.ResponseWriter, r *http.Request) {
	var data struct {
		ConfigId    string `json:"id"`
		UpstreamId  string `json:"upstream_id"`
		UpstreamURL string `json:"upstream_url"`
	}

	// get input data
	defer r.Body.Close()
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		http_err(w, "Data parsing error", err)
		return
	}

	// validate input
	if data.ConfigId == "" {
		http_err(w, "Server identifier is not specified", err)
		return
	}

	if data.UpstreamId == "" {
		http_err(w, "Upstream identifier is not specified", err)
		return
	}

	// update config with the new upstream
	config, ok := app.Configs[data.ConfigId]
	if !ok {
		http_err(w, fmt.Sprintf("Config not found: %q", data.ConfigId), nil)
		return
	}

	upstreams, ok := config.Upstreams[data.UpstreamId]
	if !ok {
		http_err(w, fmt.Sprintf("Upstream not found: %q", data.UpstreamId), nil)
		return
	}

	// add upstream
	duplicateFound := false
	for _, s := range upstreams {
		if s == data.UpstreamURL {
			duplicateFound = true
			break
		}
	}

	if !duplicateFound {
		config.Upstreams[data.UpstreamId] = append(upstreams, data.UpstreamURL)
	}

	// reconfigure nginx
	if err := app.reconfigureNginx(); err != nil {
		http_err(w, "Could not reconfigure nginx", err)
		return
	}
}

func (app *Application) DeleteUpstream(w http.ResponseWriter, r *http.Request) {
	var data struct {
		ConfigId    string `json:"id"`
		UpstreamId  string `json:"upstream_id"`
		UpstreamURL string `json:"upstream_url"`
	}

	// get input data
	defer r.Body.Close()
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		http_err(w, "Data parsing error", err)
		return
	}

	// validate config id
	if data.ConfigId == "" {
		http_err(w, "Server identifier is not specified", err)
		return
	}

	// update config with the new upstream
	config, ok := app.Configs[data.ConfigId]
	if !ok {
		http_err(w, fmt.Sprintf("Config not found: %q", data.ConfigId), nil)
		return
	}

	upstreams, ok := config.Upstreams[data.UpstreamId]
	if !ok {
		http_err(w, fmt.Sprintf("Upstream not found: %q", data.UpstreamId), nil)
		return
	}

	// check for empty upstream
	if len(upstreams) <= 0 {
		http_err(w, fmt.Sprintf("Trying to delete from empty upstream: %q", data.UpstreamId), nil)
		return
	}

	// delete upstream
	newServers := make([]string, len(upstreams)-1)
	for _, s := range upstreams {
		if s != data.UpstreamURL {
			newServers = append(newServers)
		}
	}
	config.Upstreams[data.UpstreamId] = newServers

	// reconfigure nginx
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
	http.HandleFunc("/addUpstream", app.AddUpstream)
	http.HandleFunc("/deleteUpstream", app.DeleteUpstream)

	fail_on_err(http.ListenAndServe(":3456", nil))
}
