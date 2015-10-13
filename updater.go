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
	Config struct {
		ServerConfigs   map[string]*ServerConfig
		UpstreamConfigs map[string]*UpstreamConfig
	}
	TemplateName string
	Template     *template.Template
	OutputFile   string
}

func (app *Application) Setup() {
	var err error

	app.Template, err = template.ParseFiles(app.TemplateName)
	fail_on_err(err)

	app.Config.ServerConfigs = make(map[string]*ServerConfig)
	app.Config.UpstreamConfigs = make(map[string]*UpstreamConfig)
}

func (app *Application) reconfigureNginx() error {
	f, err := os.Create(app.OutputFile)
	if err != nil {
		return err
	}
	defer f.Close()

	app.Template.Execute(f, app.Config)
	return nil
}

func (app *Application) DeleteServer(w http.ResponseWriter, r *http.Request) {
	serverId := r.FormValue("id")

	if serverId == "" {
		http_err(w, "Server identifier is not specified", nil)
		return
	}

	// delete config from map
	delete(app.Config.ServerConfigs, serverId)

	// reconfigure nginx
	if err := app.reconfigureNginx(); err != nil {
		http_err(w, "Could not reconfigure nginx", err)
		return
	}
}

func (app *Application) UpdateServer(w http.ResponseWriter, r *http.Request) {
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
	app.Config.ServerConfigs[config.Id] = &config

	// reconfigure nginx
	if err := app.reconfigureNginx(); err != nil {
		http_err(w, "Could not reconfigure nginx", err)
		return
	}
}

func (app *Application) DeleteUpstream(w http.ResponseWriter, r *http.Request) {
	upstreamId := r.FormValue("id")

	if upstreamId == "" {
		http_err(w, "Upstream identifier is not specified", nil)
		return
	}

	// delete upstream from map
	delete(app.Config.UpstreamConfigs, upstreamId)

	// reconfigure nginx
	if err := app.reconfigureNginx(); err != nil {
		http_err(w, "Could not reconfigure nginx", err)
		return
	}
}

func (app *Application) UpdateUpstream(w http.ResponseWriter, r *http.Request) {
	var upstream UpstreamConfig

	// get input
	defer r.Body.Close()
	err := json.NewDecoder(r.Body).Decode(&upstream)
	if err != nil {
		http_err(w, "Data parsing error", err)
		return
	}

	// validate input
	if upstream.Id == "" {
		http_err(w, "Upstream identifier is not specified", err)
		return
	}

	// update config
	app.Config.UpstreamConfigs[upstream.Id] = &upstream

	// reconfigureNginx
	if err := app.reconfigureNginx(); err != nil {
		http_err(w, "Could not reconfigure nginx", err)
	}
}

func (app *Application) AddUpstreamServer(w http.ResponseWriter, r *http.Request) {
	var data struct {
		UpstreamId string `json:"upstream_id"`
		ServerURL  string `json:"server_url"`
	}

	// get input data
	defer r.Body.Close()
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		http_err(w, "Data parsing error", err)
		return
	}

	// validate input
	if data.UpstreamId == "" {
		http_err(w, "Upstream identifier is not specified", err)
		return
	}

	// update config with the new upstream
	upstream, ok := app.Config.UpstreamConfigs[data.UpstreamId]
	if !ok {
		http_err(w, fmt.Sprintf("Upstream not found: %q", data.UpstreamId), nil)
		return
	}

	// add upstream
	duplicateFound := false
	for _, s := range upstream.Servers {
		if s == data.ServerURL {
			duplicateFound = true
			break
		}
	}

	if !duplicateFound {
		upstream.Servers = append(upstream.Servers, data.ServerURL)
	}

	// reconfigure nginx
	if err := app.reconfigureNginx(); err != nil {
		http_err(w, "Could not reconfigure nginx", err)
		return
	}
}

func (app *Application) DeleteUpstreamServer(w http.ResponseWriter, r *http.Request) {
	var data struct {
		UpstreamId string `json:"upstream_id"`
		ServerURL  string `json:"server_url"`
	}

	// get input data
	defer r.Body.Close()
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		http_err(w, "Data parsing error", err)
		return
	}

	// validate config id
	if data.UpstreamId == "" {
		http_err(w, "Upstream identifier is not specified", err)
		return
	}

	// get upstream
	upstream, ok := app.Config.UpstreamConfigs[data.UpstreamId]
	if !ok {
		http_err(w, fmt.Sprintf("Upstream not found: %q", data.UpstreamId), nil)
		return
	}

	// check for empty upstream
	if len(upstream.Servers) <= 0 {
		http_err(w, fmt.Sprintf("Trying to delete from empty upstream: %q", data.UpstreamId), nil)
		return
	}

	// delete upstream
	newServers := make([]string, len(upstream.Servers)-1)
	for _, s := range upstream.Servers {
		if s != data.ServerURL {
			newServers = append(newServers)
		}
	}
	upstream.Servers = newServers

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

	// server configs
	http.HandleFunc("/updateServer", app.UpdateServer)
	http.HandleFunc("/deleteServer", app.DeleteServer)

	// upstreams
	http.HandleFunc("/updateUpstream", app.UpdateUpstream)
	http.HandleFunc("/deleteUpstream", app.DeleteUpstream)
	http.HandleFunc("/addUpstreamServer", app.AddUpstreamServer)
	http.HandleFunc("/deleteUpstreamServer", app.DeleteUpstreamServer)

	fail_on_err(http.ListenAndServe(":3456", nil))
}
