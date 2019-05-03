package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"regexp"

	"github.com/airtonGit/filelogger"
	"github.com/airtonGit/version"
	"github.com/joho/godotenv"
)

func serveReverseProxy(target string, res http.ResponseWriter, req *http.Request) {
	//parse the url
	url, err := url.Parse(target)
	if err != nil {
		filelogger.Error("forwardMicroservice url.Parse:", err)
	}

	filelogger.Info("serveReverseProxy url", url)

	//create de reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(url)

	//Update the headers to allow for SSL redirection
	req.URL.Host = url.Host
	req.URL.Scheme = url.Scheme
	req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))
	req.Host = url.Host

	// Note that ServeHttp is non blocking and uses a go routine under the hood
	proxy.ServeHTTP(res, req)
}

func handlerSwitch(res http.ResponseWriter, req *http.Request) {
	fmt.Println("handlerSwitch", req.URL.Path)

	//ENDPOINTS_NAMES
	//ENDPOINTS_PATHS
	//ENDPOINTS

	type lista []string

	var endpointNames, endpointsPaths, endpoints lista

	if err := json.Unmarshal([]byte(os.Getenv("ENDPOINTS_NAMES")), &endpointNames); err != nil {
		erroMsg := "Erro ao decode json ENDPOINT_NAMES\n"
		filelogger.Error(erroMsg, err)
		fmt.Printf(erroMsg)
	}
	if err := json.Unmarshal([]byte(os.Getenv("ENDPOINTS_PATHS")), &endpointsPaths); err != nil {
		erroMsg := "Erro ao decode json ENDPOINT_PATHS\n"
		filelogger.Error(erroMsg, err)
		fmt.Printf(erroMsg)
	}
	if err := json.Unmarshal([]byte(os.Getenv("ENDPOINTS")), &endpoints); err != nil {
		erroMsg := "Erro ao decode json ENDPOINT\n"
		filelogger.Error(erroMsg, err)
		fmt.Printf(erroMsg)
	}

	fmt.Println("ENDPOINTS", endpointNames, endpointsPaths, endpoints)

	//Iterar endpoints names e acessar o index das demais
	for i, nome := range endpointNames {
		fmt.Printf("Falha ao compilar regexp path %s \n", nome)
		//Tentar fazer match em cada item
		if endpointsPaths[i] != "" {
			regexpString := fmt.Sprintf(`\%s(.*)`, endpointsPaths[i])
			re, err := regexp.Compile(regexpString)
			if err != nil {
				fmt.Printf("Falha ao compilar regexp path %s, erro %s\n", regexpString, err)
			}
			if re.Match([]byte(req.URL.Path)) {
				//Proxyed
				filelogger.Info("Encaminhando para ", nome, req.URL.Path)
				fmt.Printf("Encaminhando para %s %s", nome, req.URL.Path)
				serveReverseProxy(endpoints[i], res, req)
				break
			}
		} else {
			// path vazio, encaminhar
			filelogger.Info("Encaminhando para ", nome, req.URL.Path)
			fmt.Printf("Encaminhando para %s %s", nome, req.URL.Path)
			serveReverseProxy(endpoints[i], res, req)
			break
		}
	}
}

//Version é atualizada pela opcao do linker -X main.Version=1.5
var Version = "Versão não informada"

func main() {

	if err := godotenv.Load(); err != nil {
		log.Println("File .env not found, reading configuration from ENV")
	}

	var (
		listenPort string
		serverCert string
		serverKey  string
		logfile    string
		tlsOption  bool
	)

	//jogando logs na tela também

	//log.SetOutput(os.Stdout)
	//log.SetOutput()

	flag.StringVar(&serverCert, "cert", "cert.pem", "Informar o caminho do arquivo do certificado")
	flag.StringVar(&serverKey, "key", "key.pem", "Informar o arquivo key")
	flag.StringVar(&logfile, "logfile", "reverse-proxy.log", "Informe caminho completo com nome do arquivo de log")
	flag.BoolVar(&tlsOption, "tls", false, "Habilitar servidor https porta 443")
	flag.StringVar(&listenPort, "p", "80", "Informe porta tcp, onde aguarda requisições, padrão 80")

	//version.ParseAll("0.8")
	version.ParseAll(Version)

	filelogger.StartLogWithTag(logfile, "reverse-proxy ")
	filelogger.Info("Iniciando reverse-proxy na porta ", listenPort)

	http.HandleFunc("/", handlerSwitch)

	if tlsOption {
		go func() {
			filelogger.Info("TLS https server enabled")
			startHTTPSServer(serverCert, serverKey)
		}()
	} else {
		filelogger.Info("TLS https server off")
	}

	filelogger.Info("Iniciando proxy porta ", listenPort)
	if err := http.ListenAndServe(":"+listenPort, nil); err != nil {
		filelogger.Error("Servidor Http:80 erro:", err)
	}
}

func startHTTPSServer(serverCert string, serverKey string) {
	if _, err := os.Open(serverCert); err != nil {
		filelogger.Error("Falha ao abrir Cert arquivo, encerrando.")
		os.Exit(1)
	}

	if _, err := os.Open(serverKey); err != nil {
		filelogger.Error("Falha ao abrir Key arquivo, encerrando.")
		os.Exit(1)
	}

	filelogger.Info("Iniciando proxy porta 443")
	if err := http.ListenAndServeTLS(":443", serverCert, serverKey, nil); err != nil {
		filelogger.Error("Servidor Http:443 erro:", err)
	}
}
