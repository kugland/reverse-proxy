package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"regexp"

	"github.com/airtonGit/filelogger"
	"github.com/airtonGit/version"
	"github.com/joho/godotenv"
)

type reverseProxy struct {
	log *filelogger.Filelogger
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

func (r *reverseProxy) ServeHTTP(res http.ResponseWriter, req *http.Request) { //handlerSwitch
	r.log.Info("handlerSwitch", req.URL)

	//ENDPOINTS_NAMES
	//ENDPOINTS_PATHS
	//ENDPOINTS

	type lista []string

	var endpointNames, endpointsPaths, endpoints lista

	// if err := json.Unmarshal([]byte(os.Getenv("HOST_NAMES")), &hostnames); err != nil {
	// 	erroMsg := "Erro ao decode json HOST_NAMES\n"
	// 	r.log.Error(erroMsg, err.Error())
	// }

	if err := json.Unmarshal([]byte(os.Getenv("ENDPOINTS_NAMES")), &endpointNames); err != nil {
		erroMsg := "Erro ao decode json ENDPOINT_NAMES\n"
		r.log.Error(erroMsg, err.Error())
	}
	if err := json.Unmarshal([]byte(os.Getenv("ENDPOINTS_PATHS")), &endpointsPaths); err != nil {
		erroMsg := "Erro ao decode json ENDPOINT_PATHS\n"
		r.log.Error(erroMsg, err.Error())
	}
	if err := json.Unmarshal([]byte(os.Getenv("ENDPOINTS")), &endpoints); err != nil {
		erroMsg := "Erro ao decode json ENDPOINT\n"
		r.log.Error(erroMsg, err.Error())
	}

	r.log.Info("ENDPOINTS", endpointNames, endpointsPaths, endpoints)

	//Iterar endpoints names e acessar o index das demais
	requestServed := false
	for i, nome := range endpointNames {
		fmt.Printf("Falha ao compilar regexp path %s \n", nome)
		//Tentar fazer match em cada item
		if endpointsPaths[i] != "" {
			r.log.Info(fmt.Sprintf("Tentando path: %s url host: %s url path: %s, url:%s", endpointsPaths[i], req.URL.Host, req.URL.Path, req.URL.String()))
			regexpString := fmt.Sprintf(`%s(.*)`, endpointsPaths[i])
			re, err := regexp.Compile(regexpString)
			if err != nil {
				//r.log.Info(fmt.Sprintf("Falha ao compilar regexp path %s, erro %s\n", regexpString, err.Error()))
				continue
			}
			if re.Match([]byte(req.URL.String())) {
				//Proxyed
				r.log.Info("Encaminhando para ", nome, endpoints[i], req.URL.Path)
				r.serveReverseProxy(endpoints[i], res, req)
				requestServed = true
				break //Encontrei handler, 1o encontrado 1o atende
			}
		}
	}
	if requestServed == false {
		r.log.Warning(fmt.Sprintf("Request não atendido host: %s url path: %s, url:%s", req.URL.Host, req.URL.Path, req.URL.String()))
	}
}

//Version é atualizada pela opcao do linker -X main.Version=1.5
var Version = "Versão não informada"

func main() {

	var logfile string
	flag.StringVar(&logfile, "logfile", "reverse-proxy.log", "Informe caminho completo com nome do arquivo de log")

	log, err := filelogger.New(logfile, "reverse-proxy ")
	if err != nil {
		panic(fmt.Sprintf("Não foi possivel iniciar logger info:%s", err.Error()))
	}

	if err := godotenv.Load(); err != nil {
		log.Error("Arquivo .env indisponivel, configuracao de variaveis ENV")
	}

	var listenPort, serverCert, serverKey string
	var tlsOption bool

	flag.StringVar(&serverCert, "cert", os.Getenv("CERT"), "Informar o caminho do arquivo do certificado")
	flag.StringVar(&serverKey, "key", os.Getenv("KEY"), "Informar o arquivo key")
	flag.BoolVar(&tlsOption, "tls", os.Getenv("TLS") == "true", "Habilitar servidor https porta 443")
	flag.StringVar(&listenPort, "p", os.Getenv("PORT"), "Informe porta tcp, onde aguarda requisições, padrão 80")

	version.ParseAll(Version)

	log.Info("Iniciando reverse-proxy na porta ", listenPort)
	reverseProxy := &reverseProxy{log}
	http.Handle("/", reverseProxy)

	if tlsOption {
		go func() {
			log.Info("TLS https server enabled")
			reverseProxy.startHTTPSServer(serverCert, serverKey)
		}()
	} else {
		log.Info("TLS https server off")
	}

	if err := http.ListenAndServe(fmt.Sprintf(":%s", listenPort), nil); err != nil {
		log.Fatal("Servidor Http:80 erro:", err.Error())
	}
}

func (r *reverseProxy) startHTTPSServer(serverCert string, serverKey string) {
	if _, err := os.Open(serverCert); err != nil {
		r.log.Fatal("Falha ao abrir Cert arquivo, encerrando.")
	}

	if _, err := os.Open(serverKey); err != nil {
		r.log.Fatal("Falha ao abrir Key arquivo, encerrando.")
	}

	r.log.Info("Iniciando proxy porta 443")
	if err := http.ListenAndServeTLS(":443", serverCert, serverKey, nil); err != nil {
		r.log.Error("Servidor Http:443 erro:", err)
	}
}
