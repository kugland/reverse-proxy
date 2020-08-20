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

	log.Info("Iniciando reverse-proxy addr ", *listenAddr)
	reverseProxy := &srv.ReverseProxy{Log: log, Addr: *listenAddr}

	err = reverseProxy.LoadConfig()
	if err != nil {
		log.Fatal(err.Error())
	}

	reverseProxy.Listen()
}
