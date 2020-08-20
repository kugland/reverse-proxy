package proxy

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/airtonGit/monologger"
	"github.com/gorilla/mux"
	"gopkg.in/yaml.v2"
)

//ServerConfig representa arquivo de configuracao
type ServerConfig struct {
	List []struct {
		ServerName []string `json:"servername" yaml:"servername"`
		//Locations  []locationConfig "json:locations"
		Locations []struct {
			Path     string `json:"path" yaml:"path"`
			Endpoint string `json:"endpoint" yaml:"endpoint"`
		} `json:"locations" yaml:"locations"`
		TLS  bool   `json:"tls" yaml:"tls"`
		Cert string `json:"cert" yaml:"cert"`
		Key  string `json:"certkey" yaml:"certkey"`
	} `json:"proxy" yaml:"proxy"`
}

//ReverseProxy distribui requisicoes de acordo com os paths
type ReverseProxy struct {
	Log       *monologger.Log
	Config    ServerConfig
	DebugMode bool
	Addr      string
}

func (r *ReverseProxy) serveReverseProxy(target string, res http.ResponseWriter, req *http.Request) {
	//parse the url
	url, err := url.Parse(target)
	if err != nil {
		r.Log.Error("forwardMicroservice url.Parse:", err)
	}

	r.Log.Info("serveReverseProxy url", url)

	//create de reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(url)

	//Update the headers to allow for SSL redirection
	r.Log.Info("req.Host", req.Host)
	r.Log.Info("req.URL.host", req.URL.Host)
	r.Log.Info("Url.Host from target", url.Host)
	req.URL.Host = url.Host
	req.URL.Scheme = url.Scheme
	r.Log.Info("X-Forwarded-Host = req.Host", req.Host)
	req.Header.Set("X-Forwarded-Host", req.Host) //req.Header.Get("Host"))
	//req.Host = url.Host

	// Note that ServeHttp is non blocking and uses a go routine under the hood
	proxy.ServeHTTP(res, req)
}

//LoadConfig carrega config
func (r *ReverseProxy) LoadConfig() error {
	configYaml, err := os.Open("config.yaml")
	if err != nil {
		return fmt.Errorf("Falha o abrir config.json %s", err.Error())
	}
	defer configYaml.Close()
	if err != nil {
		return fmt.Errorf("Falha ao ler config.yaml %s", err.Error())
	}
	config := ServerConfig{}
	err = yaml.NewDecoder(configYaml).Decode(&config)
	if err != nil {
		return fmt.Errorf("Erro no arquivo config.json err:%s", err.Error())
	}
	r.Config = config
	return nil
}

func stringMatch(location, url string) (bool, error) {
	return strings.HasPrefix(url, location), nil
}

func matchURLPart(urlPart, url string) (bool, error) {
	//primeiro que dar match atende
	re, err := regexp.Compile(fmt.Sprintf("^%s.*", urlPart))
	if err != nil {
		fmt.Printf("Fail compile Regexp %s", err.Error())
		return false, err
	}
	if re.Match([]byte(url)) {
		return true, nil
	}

	return false, nil
}

//Listen serve http e https
func (r *ReverseProxy) Listen() {
	srv := http.Server{
		Addr:              r.Addr,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
	}
	hasTLS := false
	router := mux.NewRouter()
	for _, host := range r.Config.List {
		if host.TLS {
			hasTLS = true
		}
		for _, alias := range host.ServerName {
			for _, path := range host.Locations {
				sub := router.Host(alias).Subrouter()
				sub.HandleFunc(path.Path, func(w http.ResponseWriter, req *http.Request) {
					r.Log.Info("Encaminhando para ", path.Endpoint, req.URL.Path)
					r.serveReverseProxy(path.Endpoint, w, req)
				})
			}
		}
	}

	if hasTLS {
		go func() {
			r.StartHTTPSServer()
		}()
	}

	if err := srv.ListenAndServe(); err != nil {
		log.Fatal("Servidor Http erro:", err.Error())
	}

}

//StartHTTPSServer https server
func (r *ReverseProxy) StartHTTPSServer() error {

	tlsConfig := &tls.Config{}
	tlsConfig.Certificates = make([]tls.Certificate, 0)
	atLastOneTLS := false
	for _, server := range r.Config.List {
		if server.TLS == false {
			continue
		}
		atLastOneTLS = true
		if _, err := os.Open(server.Cert); err != nil {
			r.Log.Fatal("Falha ao abrir Cert arquivo, encerrando.", server.ServerName, server.Cert, err.Error())
			return err
		}

		if _, err := os.Open(server.Key); err != nil {
			r.Log.Fatal("Falha ao abrir Key arquivo, encerrando.", server.ServerName, server.Key, err.Error())
			return err
		}

		r.Log.Info("Iniciando proxy porta 443")

		// go http server treats the 0'th key as a default fallback key
		tlsKeyPair, err := tls.LoadX509KeyPair(server.Cert, server.Key)
		if err != nil {
			r.Log.Error("não pode criar par-chave", server.ServerName)
			return err
		}
		tlsConfig.Certificates = append(tlsConfig.Certificates, tlsKeyPair)
	}

	tlsConfig.BuildNameToCertificate()

	if atLastOneTLS == false {
		r.Log.Info("No one tls server setup")
		return nil
	}

	serverTLS := &http.Server{
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		//MaxHeaderBytes: 1 << 20,
		TLSConfig: tlsConfig,
	}

	listener, err := tls.Listen("tcp", ":443", tlsConfig)
	if err != nil {
		r.Log.Fatal("Https listener", err)
	}
	log.Fatal(serverTLS.Serve(listener))
	return nil
}
