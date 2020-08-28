package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/airtonGit/monologger"
	srv "github.com/airtonGit/reverse-proxy/proxy"
	"github.com/peterbourgon/ff"
)

func makeProxy(confFilename, listenAddr string, log *monologger.Log) (*srv.ReverseProxy, error) {
	proxy := &srv.ReverseProxy{Log: log, Addr: listenAddr}
	configYaml, err := os.Open(confFilename)
	if err != nil {
		return nil, err
	}
	defer configYaml.Close()
	err = proxy.LoadConfig(configYaml)
	if err != nil {
		return nil, err
	}
	return proxy, nil
}

func main() {
	fmt.Println("Reverse-Proxy 2.0.6 (pathprefix)")
	fs := flag.NewFlagSet("reverse-proxy", flag.ExitOnError)
	var (
		listenAddr    = fs.String("listen-addr", ":8080", "listen address")
		debugMode     = fs.Bool("debug", false, "log debug information")
		proxyConfFile = fs.String("proxy-conf-file", "config.yaml", "proxy rules")
		// _          = fs.String("config", "", "config file (optional)")
	)

	ff.Parse(fs, os.Args[1:],
		ff.WithEnvVarPrefix("REVERSEPROXY"),
		// ff.WithConfigFileFlag("config"),
		// ff.WithConfigFileParser(ff.PlainParser),
	)

	if *debugMode {
		log.Println("debug mode ON")
	}

	log, err := monologger.New(os.Stdout, "reverse-proxy", *debugMode)
	if err != nil {
		panic(fmt.Sprintf("Não foi possivel iniciar logger info:%s", err.Error()))
	}
	log.SetDebug(*debugMode)

	log.Info("Iniciando reverse-proxy addr ", *listenAddr)
	reverseProxy, err := makeProxy(*proxyConfFile, *listenAddr, log)
	if err != nil {
		log.Fatal(fmt.Sprintf("Falha o abrir %s %s", *proxyConfFile, err.Error()))
	}
	reverseProxy.Setup()
	reverseProxy.Listen()
}
