package main

import (
	"flag"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"regexp"

	"github.com/airtonGit/filelogger"
	"github.com/airtonGit/version"
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
	re, err := regexp.Compile(`\/api(.*)`)
	if err != nil {
		fmt.Println("Falha ao compilar regexp", err)
	}
	if re.Match([]byte(req.URL.Path)) {
		//Proxyed
		filelogger.Info("Encaminhando para api-gateway", req.URL.Path)
		serveReverseProxy("http://127.0.0.1:9000", res, req)
	} else {
		//Requisicao normal Apache
		filelogger.Info("Encaminhando req para Apache", req.URL.Path)
		serveReverseProxy("http://127.0.0.1:8080/", res, req)
	}
}

func startHTTPServer(serverCert string, serverKey string, tlsOption bool) {
	if _, err := os.Open(serverCert); tlsOption && err != nil {
		filelogger.Error("Falha ao abrir Cert arquivo, encerrando.")
		os.Exit(1)
	}

	if _, err := os.Open(serverKey); tlsOption && err != nil {
		filelogger.Error("Falha ao abrir Key arquivo, encerrando.")
		os.Exit(1)
	}

	filelogger.Info("Iniciando proxy porta 443")
	if err := http.ListenAndServeTLS(":443", serverCert, serverKey, nil); err != nil {
		filelogger.Error("Servidor Http:443 erro:", err)
	}
}

func main() {
	var (
		serverCert string
		serverKey  string
		logfile    string
		tlsOption  bool
	)

	flag.StringVar(&serverCert, "cert", "cert.pem", "Informar o caminho do arquivo do certificado")
	flag.StringVar(&serverKey, "key", "key.pem", "Informar o arquivo key")
	flag.StringVar(&logfile, "logfile", "reverse-proxy.log", "Informe caminho completo com nome do arquivo de log")
	flag.BoolVar(&tlsOption, "tls", false, "Habilitar servidor https porta 443")

	version.ParseAll("0.4")

	filelogger.StartLogWithTag(logfile, "reverse-proxy ")
	filelogger.Info("Iniciando reverse-proxy")

	http.HandleFunc("/", handlerSwitch)

	if tlsOption {
		go func() {
			filelogger.Info("TLS https server enabled")
			startHTTPServer(serverCert, serverKey, tlsOption)
		}()
	} else {
		filelogger.Info("TLS https server off")
	}

	filelogger.Info("Iniciando proxy porta 80")
	if err := http.ListenAndServe(":80", nil); err != nil {
		filelogger.Error("Servidor Http:80 erro:", err)
	}
}
