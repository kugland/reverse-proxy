package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/airtonGit/monologger"
	"github.com/joho/godotenv"
)

// matchURLPart(urlPart, url string) bool {)
func TestMatchURLPart(t *testing.T) {
	got, err := matchURLPart("/dashboard", "/api/dashboard/alguma/coisa")
	if err != nil {
		t.Error("Falha", err.Error())
	}
	if got == false {
		t.Error("Não pode dar match em /")
	}
}

func TestStringMatch(t *testing.T) {
	got, err := stringMatch("/", "/api/dashboard/alguma/coisa")
	if err != nil {
		t.Error("Falha", err.Error())
	}
	if got == false {
		t.Error("Não pode dar match em /")
	}
}

func TestMain(t *testing.T) {
	destinoArq, err := os.OpenFile("test-logfile.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		t.Error("Não pode criar/abrir log file", err.Error())
	}
	log, err := monologger.New(destinoArq, "reverse-proxy", true)
	if err != nil {
		t.Error("Não pode criar log file")
	}

	if err := godotenv.Load(); err != nil {
		log.Error("Arquivo .env indisponivel, configuracao de variaveis ENV")
	}

	reverseProxy := &reverseProxy{log: log}

	var listaReq []*http.Request

	listaReq = append(listaReq, httptest.NewRequest("POST", "http://www.site.com.br/", nil))
	listaReq = append(listaReq, httptest.NewRequest("POST", "http://site.com.br/", nil))

	//assert.NoError(t, errReq, "Falhou NewRequest")

	//We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	for _, req := range listaReq {
		reverseProxy.ServeHTTP(rr, req)
	}

}
