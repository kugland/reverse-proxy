package proxy

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"time"

	"github.com/airtonGit/monologger"
	"github.com/gorilla/mux"
	"gopkg.in/yaml.v2"
)

//ProxyItem grupo de dominios com mesmo certificado TLS e paths
type ProxyItem struct {
	ServerName []string `json:"servername" yaml:"servername"`
	//Locations  []locationConfig "json:locations"
	Locations []struct {
		Path     string `json:"path" yaml:"path"`
		Endpoint string `json:"endpoint" yaml:"endpoint"`
	} `json:"locations" yaml:"locations"`
	TLS  bool   `json:"tls" yaml:"tls"`
	Cert string `json:"cert" yaml:"cert"`
	Key  string `json:"certkey" yaml:"certkey"`
}

//ServerConfig representa arquivo de configuracao
type ServerConfig struct {
	List []ProxyItem `json:"proxy" yaml:"proxy"`
}

//ReverseProxy distribui requisicoes de acordo com os paths
type ReverseProxy struct {
	Log       *monologger.Log
	Config    ServerConfig
	DebugMode bool
	Addr      string
	Srv       *http.Server
	TLS       bool
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
	req.Header.Set("X-Forwarded-Host", req.Host)

	// Note that ServeHttp is non blocking and uses a go routine under the hood
	proxy.ServeHTTP(res, req)
}

//LoadConfig carrega config
func (r *ReverseProxy) LoadConfig(configYaml io.ReadCloser) error {
	config := ServerConfig{}
	err := yaml.NewDecoder(configYaml).Decode(&config)
	if err != nil {
		return fmt.Errorf("Erro no arquivo config.json err:%s", err.Error())
	}
	r.Config = config
	return nil
}

func (r *ReverseProxy) makeHandler(path, endpoint string) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		r.Log.Info("Encaminhando para ", path, endpoint, req.URL.Path)
		r.serveReverseProxy(endpoint, w, req)
	}
}

//Setup serve http e https
func (r *ReverseProxy) Setup() {

	router := mux.NewRouter()
	for _, host := range r.Config.List {
		if host.TLS {
			r.TLS = true
		}
		for _, alias := range host.ServerName {
			sub := router.Host(alias).Subrouter()
			for _, path := range host.Locations {
				r.Log.Debug("Adicionando regra", alias, path.Path)
				sub.PathPrefix(path.Path).HandlerFunc(r.makeHandler(path.Path, path.Endpoint))
			}
		}
	}
	r.Srv = &http.Server{
		Handler:           router,
		Addr:              r.Addr,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
	}
}

func (r *ReverseProxy) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	r.Srv.Handler.ServeHTTP(res, req)
}

//Listen aguarda clientes, precisa ser configurado com Setup
func (r *ReverseProxy) Listen() {
	if r.TLS {
		go func() {
			err := r.StartHTTPSServer()
			if err != nil {
				r.Log.Error("Https Server error", err)
			}
		}()
	}

	if err := r.Srv.ListenAndServe(); err != nil {
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
		Handler:      r.Srv.Handler,
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
