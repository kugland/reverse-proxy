package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"regexp"
	"time"

	"github.com/airtonGit/filelogger"
	"github.com/joho/godotenv"
)

type serverConfig struct {
	ServerName []string `json:"servername"`
	//Locations  []locationConfig "json:locations"
	Locations []struct {
		Path     string `json:"path"`
		Endpoint string `json:"endpoint"`
	} `json:"locations"`
	TLS  bool   `json:"tls"`
	Cert string `json:"cert"`
	Key  string `json:"key"`
}

type reverseProxy struct {
	log    *filelogger.Filelogger
	Config []serverConfig
}

func (r *reverseProxy) serveReverseProxy(target string, res http.ResponseWriter, req *http.Request) {
	//parse the url
	url, err := url.Parse(target)
	if err != nil {
		r.log.Error("forwardMicroservice url.Parse:", err)
	}

	r.log.Info("serveReverseProxy url", url)

	//create de reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(url)

	//Update the headers to allow for SSL redirection
	//req.URL.Host = url.Host
	req.URL.Scheme = url.Scheme
	req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))
	//req.Host = url.Host

	// Note that ServeHttp is non blocking and uses a go routine under the hood
	proxy.ServeHTTP(res, req)
}

func (r *reverseProxy) loadConfig() error {
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

func matchURLPart(urlPart, url string) bool {
	//primeiro que dar match atende
	re, err := regexp.Compile(fmt.Sprintf(`%s(.*)`, urlPart))
	if err != nil {
		//r.log.Info(fmt.Sprintf("Falha ao compilar regexp urlPart %s, url %s, erro %s\n", urlPart, url, err.Error()))
		return false
	}
	if re.Match([]byte(url)) {
		return true
	}

	return false
}

func (r *reverseProxy) ServeHTTP(res http.ResponseWriter, req *http.Request) { //handlerSwitch
	r.log.Info("handlerSwitch", req.URL)
	//r.log.SetDebug(true)
	//Iterar endpoints names e acessar o index das demais
	requestServed := false
	for _, server := range r.Config {
		//Cada server config pode ter alguns subdominios www.dominio.com ou dominio.com
		for _, serverName := range server.ServerName {
			r.log.Debug("Tentando config", serverName, "requisicao ", req.URL.Hostname())
			if true == matchURLPart(serverName, req.URL.Hostname()) {
				//Domain found, match location
				for _, location := range server.Locations {
					r.log.Debug("Tentando location", location.Path, "requisicao ", req.URL.Path)
					if true == matchURLPart(location.Path, req.URL.Path) {
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

func (r *reverseProxy) startHTTPSServer() {

	tlsConfig := &tls.Config{}
	tlsConfig.Certificates = make([]tls.Certificate, 1)
	atLastOneTLS := false
	for _, server := range r.Config {
		if server.TLS == false {
			continue
		}
		atLastOneTLS = true
		if _, err := os.Open(server.Cert); err != nil {
			r.log.Fatal("Falha ao abrir Cert arquivo, encerrando.")
		}

		if _, err := os.Open(server.Key); err != nil {
			r.log.Fatal("Falha ao abrir Key arquivo, encerrando.")
		}

		r.log.Info("Iniciando proxy porta 443")

		// go http server treats the 0'th key as a default fallback key
		tlsKeyPair, err := tls.LoadX509KeyPair(server.Cert, server.Key)
		if err != nil {
			r.log.Error("não pode criar par-chave", server.ServerName)
		}
		tlsConfig.Certificates = append(tlsConfig.Certificates, tlsKeyPair)

		tlsConfig.BuildNameToCertificate()
	}

	if atLastOneTLS == false {
		r.log.Info("No one tls server setup")
		return
	}

	//http.HandleFunc("/", myHandler)
	serverTLS := &http.Server{
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
		TLSConfig:      tlsConfig,
	}

	listener, err := tls.Listen("tcp", ":8443", tlsConfig)
	if err != nil {
		r.log.Fatal("Https listener", err)
	}
	log.Fatal(serverTLS.Serve(listener))
}

func main() {

	var logfile string
	flag.StringVar(&logfile, "logfile", "reverse-proxy.log", "Informe caminho completo com nome do arquivo de log")

	log, err := filelogger.New(logfile, "reverse-proxy ")
	if err != nil {
		panic(fmt.Sprintf("Não foi possivel iniciar logger info:%s", err.Error()))
	}
	log.SetDebug(true)

	if err := godotenv.Load(); err != nil {
		log.Error("Arquivo .env indisponivel, configuracao de variaveis ENV")
	}

	var listenPort string

	flag.StringVar(&listenPort, "p", os.Getenv("PORT"), "Informe porta tcp, onde aguarda requisições, padrão 80")
	flag.Parse()

	log.Info("Iniciando reverse-proxy na porta ", listenPort)
	reverseProxy := &reverseProxy{log: log}

	err = reverseProxy.loadConfig()
	if err != nil {
		log.Fatal(err.Error())
	}

	http.Handle("/", reverseProxy)

	go func() {
		log.Info("TLS https server enabled")
		reverseProxy.startHTTPSServer()
	}()

	if err := http.ListenAndServe(fmt.Sprintf(":%s", listenPort), nil); err != nil {
		log.Fatal("Servidor Http:80 erro:", err.Error())
	}
}
