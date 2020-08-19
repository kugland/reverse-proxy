package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/airtonGit/monologger"
	"github.com/joho/godotenv"
	"gopkg.in/yaml.v2"
)

func configurePaths() error {
	configYaml, err := os.Open("config.yaml")
	if err != nil {
		return fmt.Errorf("Falha o abrir config.json %s", err.Error())
	}
	defer configYaml.Close()
	configBytes, err := ioutil.ReadAll(configYaml)
	if err != nil {
		return fmt.Errorf("Falha ao ler config.yaml %s", err.Error())
	}
	config := &serverConfig{}
	if err := yaml.Unmarshal(configBytes, &r.Config); err != nil {
		erroMsg := "Erro no arquivo config.json\n"
		r.log.Error(erroMsg, err.Error())
	}
	return nil
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
	reverseProxy := &ReverseProxy{log: log}

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
