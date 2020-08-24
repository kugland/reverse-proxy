package proxy

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/airtonGit/monologger"
)

func TestServeHTTP(t *testing.T) {

	log, err := monologger.New(os.Stdout, "reverse-proxy", true)
	if err != nil {
		t.Error("Não pode criar log file", err)
	}

	reverseProxy := &ReverseProxy{Log: log}
	configYaml, err := os.Open("./../config.yaml")
	if err != nil {
		log.Error(fmt.Sprintf("Falha o abrir config.json %s", err.Error()))
	}
	defer configYaml.Close()
	err = reverseProxy.LoadConfig(configYaml)
	if err != nil {
		t.Error("Não pode carregar config", err)
	}
	reverseProxy.Setup()

	var listaReq []*http.Request

	listaReq = append(listaReq, httptest.NewRequest("POST", "http://www.exemplo.com/", nil))
	listaReq = append(listaReq, httptest.NewRequest("POST", "http://www.exemplo.com/testepath", nil))
	listaReq = append(listaReq, httptest.NewRequest("POST", "http://outroexemplo.com.br/", nil))
	listaReq = append(listaReq, httptest.NewRequest("POST", "http://outroexemplo.com.br/relatorio", nil))

	//We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	for _, req := range listaReq {
		reverseProxy.ServeHTTP(rr, req)
	}

}
