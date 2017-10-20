package cruisecontrol

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientUtil_callCruiseControl_removeBroker(t *testing.T) {

	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Println(r)
		if r.Method != "POST" {
			t.Errorf("Expected 'POST' request, got ‘%s’", r.Method)
		}
		if r.URL.EscapedPath() != "/kafkacruisecontrol/remove_broker" {
			t.Errorf("Expected request to '/kafkacruisecontrol/remove_broker', got ‘%s’", r.URL.EscapedPath())
		}
		r.ParseForm()
		brokerID := r.Form.Get("brokerid")
		if brokerID != "1" {
			t.Errorf("Expected request to have brokerid=1’, got: ‘%s’", brokerID)
		}
		dryrun := r.Form.Get("dryrun")
		if dryrun != "false" {
			t.Errorf("Expected request to have dryrun=false’, got: ‘%s’", dryrun)
		}
	}))
	defer testServer.Close()
	requestUrl := testServer.URL

	values := map[string]string{
		"brokerid": "1",
		"dryrun":   "false",
	}
	
	CallCruiseControl(requestUrl, "remove_broker", values)

}
