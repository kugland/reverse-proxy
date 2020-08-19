package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/airtonGit/monologger"
	srv "github.com/airtonGit/reverse-proxy/service"
	"github.com/peterbourgon/ff"
	"gopkg.in/yaml.v2"
)

//ConfigurePaths carrega yaml
func ConfigurePaths() (srv.ServerConfig, error) {
	configYaml, err := os.Open("config.yaml")
	if err != nil {
		return srv.ServerConfig{}, fmt.Errorf("Falha o abrir config.json %s", err.Error())
	}
	defer configYaml.Close()
	if err != nil {
		return srv.ServerConfig{}, fmt.Errorf("Falha ao ler config.yaml %s", err.Error())
	}
	config := srv.ServerConfig{}
	err = yaml.NewDecoder(configYaml).Decode(&config)
	if err != nil {
		return srv.ServerConfig{}, fmt.Errorf("Erro no arquivo config.json err:%s", err.Error())
	}
	return config, nil
}

func main() {

	fs := flag.NewFlagSet("my-program", flag.ExitOnError)
	var (
		listenAddr = fs.String("listen-addr", "localhost:8080", "listen address")
		debugMode  = fs.Bool("debug", false, "log debug information")
		// _          = fs.String("config", "", "config file (optional)")
	)

	ff.Parse(fs, os.Args[1:],
		ff.WithEnvVarPrefix("REVERSEPROXY"),
		// ff.WithConfigFileFlag("config"),
		// ff.WithConfigFileParser(ff.PlainParser),
	)

	if !*debugMode {
		log.Println("modo debug off, REVERSEPROXY_DEBUG=true env var to activate.")
	} else {
		log.Println("modo debug ON")
	}

	log, err := monologger.New(os.Stdout, "reverse-proxy", *debugMode)
	if err != nil {
		panic(fmt.Sprintf("Não foi possivel iniciar logger info:%s", err.Error()))
	}
	log.SetDebug(true)

	log.Info("Iniciando reverse-proxy addr ", listenAddr)
	reverseProxy := &srv.ReverseProxy{Log: log}

	// err = reverseProxy.loadConfig()
	// if err != nil {
	// 	log.Fatal(err.Error())
	// }

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
			//reverseProxy.startHTTPSServer()
		}()
	}

	if err := http.ListenAndServe(fmt.Sprintf("%s", *listenAddr), nil); err != nil {
		log.Fatal("Servidor Http erro:", err.Error())
	}
}
