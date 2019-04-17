package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"regexp"

	"github.com/airtonGit/filelogger"
)

func serveReverseProxy(target string, res http.ResponseWriter, req *http.Request) {
	//parse the url
	url, _ := url.Parse(target)

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
	re, err := regexp.Compile(`\/proxy\/teste(.*)`)
	if err != nil {
		fmt.Println("Falha ao compilar regexp", err)
	}
	if re.Match([]byte(req.URL.Path)) {
		//Proxyed
		log.Println("Encaminhando para microservico", req.URL.Path)
		serveReverseProxy("http://127.0.0.1:8181", res, req)
	} else {
		//Requisicao normal Apache
		log.Println("Encaminhando req para Apache", req.URL.Path)
		serveReverseProxy("http://127.0.0.1:9090/", res, req)
	}
	//fmt.Println("Match regexp", re.FindSubmatch([]byte(req.URL.Path)))
}

func main() {
	filelogger.IniciaLogWithTag("reverse-proxy ")
	filelogger.Info("Iniciando reverse-proxy")

	http.HandleFunc("/", handlerSwitch)

	go func() {
		filelogger.Info("Iniciando proxy porta 80")
		if err := http.ListenAndServe(":80", nil); err != nil {
			filelogger.Error("Servidor Http:80 erro:", err)
		}
	}()

	var (
		serverCert string
		serverKey  string
	)

	flag.StringVar(&serverCert, "cert", "cert.pem", "Informar o caminho do arquivo do certificado")
	flag.StringVar(&serverKey, "key", "key.pem", "Informar o arquivo key")
	flag.Parse()

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
