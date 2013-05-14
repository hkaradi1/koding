package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

type ConfigFile struct {
	Mongo string
	Mq    struct {
		Host          string
		Port          int
		ComponentUser string
		Password      string
		Vhost         string
	}
}

type ServerInfo struct {
	BuildNumber string
	GitBranch   string
	GitCommit   string
	ConfigUsed  string
	Config      ConfigFile
	Hostname    Hostname
	IP          IP
}

type Hostname struct {
	Public string
	Local  string
}

type IP struct {
	Public string
	Local  string
}

type JenkinsInfo struct {
	LastCompletedBuild struct {
		Number int    `json:"number"`
		Url    string `json:"url"`
	} `json:"lastCompletedBuild"`
	LastStableBuild struct {
		Number int    `json:"number"`
		Url    string `json:"url"`
	} `json:"lastStableBuild"`
	LastFailedBuild struct {
		Number int    `json:"number"`
		Url    string `json:"url"`
	} `json:"lastFailedBuild"`
}

type WorkerInfo struct {
	Name      string    `json:"name"`
	Uuid      string    `json:"uuid"`
	Hostname  string    `json:"hostname"`
	Version   int       `json:"version"`
	Timestamp time.Time `json:"timestamp"`
	Pid       int       `json:"pid"`
	State     string    `json:"state"`
	Info      string    `json:"info"`
	Uptime    int       `json:"uptime"`
	Port      int       `json:"port"`
}

type StatusInfo struct {
	BuildNumber string
	Workers     struct {
		Running int
		Dead    int
	}
	MongoLogin string
}

type HomePage struct {
	Status  StatusInfo
	Workers []WorkerInfo
	Jenkins *JenkinsInfo
	Server  *ServerInfo
}

func NewServerInfo() *ServerInfo {
	return &ServerInfo{
		BuildNumber: "",
		GitBranch:   "",
		GitCommit:   "",
		ConfigUsed:  "",
		Config:      ConfigFile{},
		Hostname:    Hostname{},
		IP:          IP{},
	}
}

var templates = template.Must(template.ParseFiles("index.html"))

func main() {
	http.HandleFunc("/", viewHandler)
	http.Handle("/bootstrap/", http.StripPrefix("/bootstrap/", http.FileServer(http.Dir("bootstrap/"))))

	fmt.Println("koding overview started")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println(err)
	}
}

func viewHandler(w http.ResponseWriter, r *http.Request) {
	build := r.FormValue("searchbuild")
	if build == "" {
		build = "latest"
	}

	var workers []WorkerInfo
	var server *ServerInfo
	var err error

	workers, err = workerInfo(build)
	if err != nil {
		fmt.Println(err)
	}

	status := statusInfo()
	jenkins := jenkinsInfo()

	server, err = serverInfo(build)
	if err != nil {
		fmt.Println(err)
		server = NewServerInfo()
	}

	status.MongoLogin = mongoLogin(server.Config.Mongo)

	for i, val := range workers {
		switch val.State {
		case "running":
			status.Workers.Running++
			workers[i].Info = "success"
		case "dead":
			status.Workers.Dead++
			workers[i].Info = "error"
		case "stopped":
			workers[i].Info = "warning"
		case "waiting":
			workers[i].Info = "info"
		}
	}

	home := HomePage{
		Status:  status,
		Workers: workers,
		Jenkins: jenkins,
		Server:  server,
	}

	renderTemplate(w, "index", home)
	return
}

func renderTemplate(w http.ResponseWriter, tmpl string, home HomePage) {
	err := templates.ExecuteTemplate(w, tmpl+".html", home)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func jenkinsInfo() *JenkinsInfo {
	fmt.Println("getting jenkins info")
	j := &JenkinsInfo{}
	jenkinsApi := "http://salt-master.in.koding.com/job/build-koding/api/json"
	resp, err := http.Get(jenkinsApi)
	if err != nil {
		fmt.Println(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
	}

	err = json.Unmarshal(body, &j)
	if err != nil {
		fmt.Println(err)
	}

	return j
}

func statusInfo() StatusInfo {
	s := StatusInfo{}
	return s
}

func workerInfo(build string) ([]WorkerInfo, error) {
	fmt.Println("getting worker info")
	workersApi := "http://kontrol.in.koding.com/workers?version=" + build
	resp, err := http.Get(workersApi)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	workers := make([]WorkerInfo, 0)
	err = json.Unmarshal(body, &workers)
	if err != nil {
		return nil, err
	}

	return workers, nil
}

func serverInfo(build string) (*ServerInfo, error) {
	fmt.Println("getting server info")
	serverApi := "http://kontrol.in.koding.com/deployments/" + build
	fmt.Println(serverApi)

	resp, err := http.Get(serverApi)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	s := &ServerInfo{}
	err = json.Unmarshal(body, &s)
	if err != nil {
		return nil, err
	}

	return s, nil
}

func mongoLogin(login string) string {
	u, err := url.Parse("http://" + login)
	if err != nil {
		fmt.Println(err)
	}

	mPass, _ := u.User.Password()
	return fmt.Sprintf(
		"mongo %s%s -u%s -p%s",
		u.Host,
		u.Path,
		u.User.Username(),
		mPass,
	)
}
