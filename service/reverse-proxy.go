package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/airtonGit/monologger"
)

type serverConfig struct {
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

type ReverseProxy struct {
	log       *monologger.Log
	Config    []serverConfig
	DebugMode bool
}

func (r *ReverseProxy) serveReverseProxy(target string, res http.ResponseWriter, req *http.Request) {
	//parse the url
	url, err := url.Parse(target)
	if err != nil {
		r.log.Error("forwardMicroservice url.Parse:", err)
	}

	r.log.Info("serveReverseProxy url", url)

	//create de reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(url)

	//Update the headers to allow for SSL redirection
	r.log.Info("req.Host", req.Host)
	r.log.Info("req.URL.host", req.URL.Host)
	r.log.Info("Url.Host from target", url.Host)
	req.URL.Host = url.Host
	req.URL.Scheme = url.Scheme
	r.log.Info("X-Forwarded-Host = req.Host", req.Host)
	req.Header.Set("X-Forwarded-Host", req.Host) //req.Header.Get("Host"))
	//req.Host = url.Host

	// Note that ServeHttp is non blocking and uses a go routine under the hood
	proxy.ServeHTTP(res, req)
}

func (r *ReverseProxy) loadConfig() error {
	configFile, err := os.Open("config.json")
	if err != nil {
		return fmt.Errorf("Falha o abrir config.json %s", err.Error())
	}
	configBytes, err := ioutil.ReadAll(configFile)
	if err != nil {
		return fmt.Errorf("Falha ao ler config.json %s", err.Error())
	}
	configFile.Close()

	if err := json.Unmarshal(configBytes, &r.Config); err != nil {
		erroMsg := "Erro no arquivo config.json\n"
		r.log.Error(erroMsg, err.Error())
	}
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

func (r *ReverseProxy) ServeHTTP(res http.ResponseWriter, req *http.Request) { //handlerSwitch
	r.log.Info(fmt.Sprintf("http handler req.url %s, req.URL.hostname %s, req.Host %s, req.URL.Path %s", req.URL, req.URL.Hostname(), req.Host, req.URL.Path))

	//Iterar endpoints names e acessar o index das demais
	requestServed := false
	for _, server := range r.Config {
		//Cada server config pode ter alguns subdominios www.dominio.com ou dominio.com
		for _, serverName := range server.ServerName {
			r.log.Debug("Tentando config", serverName, "requisicao ", req.Host)
			if serverNameGot, _ := matchURLPart(serverName, req.Host); serverNameGot == true {
				//Domain found, match location
				for _, location := range server.Locations {
					r.log.Debug("Tentando location", location.Path, "requisicao ", req.URL.Path)
					if locationGot, _ := matchURLPart(location.Path, req.URL.Path); locationGot == true {
						r.log.Info("Encaminhando para ", location.Endpoint, req.URL.Path)
						r.serveReverseProxy(location.Endpoint, res, req)
						requestServed = true
						//break //Encontrei handler, 1o encontrado 1o atende
						return
					}
				}
			}
		}
	}
	if requestServed == false {
		r.log.Warning(fmt.Sprintf("Request não atendido host: %s url path: %s, url:%s", req.URL.Host, req.URL.Path, req.URL.String()))
	}
}

func (r *ReverseProxy) startHTTPSServer() {

	tlsConfig := &tls.Config{}
	tlsConfig.Certificates = make([]tls.Certificate, 0)
	atLastOneTLS := false
	for _, server := range r.Config {
		if server.TLS == false {
			continue
		}
		atLastOneTLS = true
		if _, err := os.Open(server.Cert); err != nil {
			r.log.Fatal("Falha ao abrir Cert arquivo, encerrando.", server.ServerName, server.Cert, err.Error())
		}

		if _, err := os.Open(server.Key); err != nil {
			r.log.Fatal("Falha ao abrir Key arquivo, encerrando.", server.ServerName, server.Key, err.Error())
		}

		r.log.Info("Iniciando proxy porta 443")

		// go http server treats the 0'th key as a default fallback key
		tlsKeyPair, err := tls.LoadX509KeyPair(server.Cert, server.Key)
		if err != nil {
			r.log.Error("não pode criar par-chave", server.ServerName)
		}
		tlsConfig.Certificates = append(tlsConfig.Certificates, tlsKeyPair)
	}

	tlsConfig.BuildNameToCertificate()

	if atLastOneTLS == false {
		r.log.Info("No one tls server setup")
		return
	}

	//http.HandleFunc("/", myHandler)
	serverTLS := &http.Server{
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		//MaxHeaderBytes: 1 << 20,
		TLSConfig: tlsConfig,
	}

	listener, err := tls.Listen("tcp", ":443", tlsConfig)
	if err != nil {
		r.log.Fatal("Https listener", err)
	}
	log.Fatal(serverTLS.Serve(listener))
}
