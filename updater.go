package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"text/template"
)

var Version = "N/D"

type Application struct {
	ConfigFile   string
	Config       *NginxConfig
	NoSaveConfig bool
	TemplateName string
	Template     *template.Template
	OutputFile   string
	NoReload     bool
}

func (app *Application) Setup() error {
	var err error

	app.Template, err = template.ParseFiles(app.TemplateName)
	if err != nil {
		return err
	}

	app.Config = NewNginxConfig()

	// try to load config from file
	if app.ConfigFile != "" {
		log.Printf("Trying to load config from %q", app.ConfigFile)
		if err := app.Config.Load(app.ConfigFile); err != nil {
			log.Printf("Couldn't load config from file. Using an empty config.")
			log.Print(err)
			// force reset config because we don't know what JSONDecoder has done to the original one
			app.Config = NewNginxConfig()
		}
	}

	return app.reconfigureNginx()
}

func (app *Application) reconfigureNginx() error {
	// try to save config to file
	if app.ConfigFile != "" && !app.NoSaveConfig {
		log.Printf("Trying to save config to %q", app.ConfigFile)
		if err := app.Config.Save(app.ConfigFile); err != nil {
			log.Printf("Couldn't save config to file. Skipping.")
			log.Print(err)
		}
	}

	// render template
	f, err := os.Create(app.OutputFile)
	if err != nil {
		return err
	}
	defer f.Close()

	app.Template.Execute(f, app.Config)

	// reload nginx
	if !app.NoReload {
		return exec.Command("nginx", "-s", "reload").Run()
	}

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
	// we assume that new array will contain (n-1) elements; so capacity=n-1
	newServers := make([]string, 0, len(upstream.Servers)-1)
	for _, s := range upstream.Servers {
		if s != data.ServerURL {
			newServers = append(newServers, s)
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
	var err error

	app := Application{}

	var doShowVersion bool
	var listenTo string

	flag.BoolVar(&doShowVersion, "version", false, "Show application version and exit")
	flag.StringVar(&app.TemplateName, "template", "default.conf.tmpl", "Config file template to be rendered. Default: default.conf.tmpl")
	flag.StringVar(&app.OutputFile, "out", "/etc/nginx/conf.d/default.conf", "Path to the config file to be updated. Default: /etc/nginx/conf.d/default.conf")
	flag.StringVar(&app.ConfigFile, "config", "", "Path to the config file. Default: empty")
	flag.BoolVar(&app.NoSaveConfig, "no-save-config", false, "Don't save (overwrite existing) config file. Default: false")
	flag.StringVar(&listenTo, "listen", ":3456", "Host and port to listen to. Default: :3456")
	flag.BoolVar(&app.NoReload, "no-reload", false, "Don't reload Nginx when applying changes. Default: false")
	flag.Parse()

	if doShowVersion {
		fmt.Println(Version)
		return
	}

	// initialize app
	err = app.Setup()
	if err != nil {
		log.Fatal(err)
	}

	// server configs
	http.HandleFunc("/updateServer", app.UpdateServer)
	http.HandleFunc("/deleteServer", app.DeleteServer)

	// upstreams
	http.HandleFunc("/updateUpstream", app.UpdateUpstream)
	http.HandleFunc("/deleteUpstream", app.DeleteUpstream)
	http.HandleFunc("/addUpstreamServer", app.AddUpstreamServer)
	http.HandleFunc("/deleteUpstreamServer", app.DeleteUpstreamServer)

	err = http.ListenAndServe(listenTo, nil)
	if err != nil {
		log.Fatal(err)
	}
}
