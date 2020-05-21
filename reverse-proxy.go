package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
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
	Key  string `json:"certkey"`
}

type reverseProxy struct {
	log       *monologger.Log
	Config    []serverConfig
	DebugMode bool
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

func (r *reverseProxy) ServeHTTP(res http.ResponseWriter, req *http.Request) { //handlerSwitch
	r.log.Info(fmt.Sprintf("http handler req.url %s, req.URL.hostname %s, req.Host %s, req.URL.Path %s", req.URL, req.URL.Hostname(), req.Host, req.URL.Path))

	// if r.DebugMode == true {
	// 	r.log.Info("Modo debug habilitado por variavel de ambiente")
	// 	r.log.SetDebug(true)
	// }
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

func (r *reverseProxy) startHTTPSServer() {

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

func main() {

	if err := godotenv.Load(); err != nil {
		fmt.Println("Arquivo .env indisponivel, configuracao de variaveis ENV")
	}

	var logfile string
	flag.StringVar(&logfile, "logfile", "", "Informe caminho completo com nome do arquivo de log")

	debugMode := os.Getenv("REVERSEPROXY_DEBUG") == "true"
	if _, got := os.LookupEnv("REVERSEPROXY_DEBUG"); got == false {
		log.Println("modo debug off, REVERSEPROXY_DEBUG=true env var to activate.")
		debugMode = false
	} else {
		log.Println("modo debug ON")
	}

	var destinoLog io.Writer
	var err error
	if logfile != "" {
		destinoLog, err = os.OpenFile(logfile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			fmt.Println("ReverseProxy - Init fail, cannot open logfile", err.Error())
			os.Exit(1)
		}
	} else {
		destinoLog = os.Stdout
	}

	log, err := monologger.New(destinoLog, "reverse-proxy", debugMode) //filelogger.NewStdoutOnly("reverse-proxy ", debugMode)
	if err != nil {
		panic(fmt.Sprintf("Não foi possivel iniciar logger info:%s", err.Error()))
	}
	log.SetDebug(true)

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

	hasTLS := false
	for _, server := range reverseProxy.Config {
		if server.TLS == true {
			hasTLS = true
			break
		}
	}
	if hasTLS {
		go func() {
			log.Info("TLS https server enabled")
			reverseProxy.startHTTPSServer()
		}()
	}

	if err := http.ListenAndServe(fmt.Sprintf(":%s", listenPort), nil); err != nil {
		log.Fatal("Servidor Http:80 erro:", err.Error())
	}
}
