package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/airtonGit/filelogger"
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
	log, err := filelogger.New("test-logfile.log", "reverse-proxy ")
	if err != nil {
		t.Error("Não pode criar log file")
	}

	if err := godotenv.Load(); err != nil {
		log.Error("Arquivo .env indisponivel, configuracao de variaveis ENV")
	}

	reverseProxy := &reverseProxy{log: log}

	var listaReq []*http.Request

	listaReq = append(listaReq, httptest.NewRequest("POST", "http://www.hotelpago.com.br/", nil))
	listaReq = append(listaReq, httptest.NewRequest("POST", "http://hotelpago.com.br/", nil))
	listaReq = append(listaReq, httptest.NewRequest("POST", "http://devel.oplen.com.br/Souza_Cruz-Projeto_Connection-Webservice/alternativo", nil))
	listaReq = append(listaReq, httptest.NewRequest("POST", "http://devel.oplen.com.br/Souza_Cruz-Projeto_Connection-Webservice", nil))
	listaReq = append(listaReq, httptest.NewRequest("POST", "http://devel.oplen.com.br/Souza_Cruz-App_Produtor_Rural-Webservice", nil))
	listaReq = append(listaReq, httptest.NewRequest("POST", "http://hml.phyllagrotech.com/", nil))

	//assert.NoError(t, errReq, "Falhou NewRequest")

	//We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	for _, req := range listaReq {
		reverseProxy.ServeHTTP(rr, req)
	}

}
