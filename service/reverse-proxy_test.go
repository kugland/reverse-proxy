package proxy

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/airtonGit/monologger"
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

	log, err := monologger.New(os.Stdout, "reverse-proxy", true)
	if err != nil {
		t.Error("Não pode criar log file")
	}

	reverseProxy := &ReverseProxy{Log: log}

	var listaReq []*http.Request

	listaReq = append(listaReq, httptest.NewRequest("POST", "http://www.site.com.br/", nil))
	listaReq = append(listaReq, httptest.NewRequest("POST", "http://site.com.br/", nil))

	//We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()

	for _, req := range listaReq {
		reverseProxy.ServeHTTP(rr, req)
	}

}
