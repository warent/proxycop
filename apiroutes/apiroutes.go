package apiroutes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/gorilla/mux"

	"github.com/warent/proxycop/utility"
)

func ConfigHandler(w http.ResponseWriter, r *http.Request) {

	fmt.Fprintln(w, "Hello api")

}

func URLStatusHandler(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	parsedURL, err := url.Parse(fmt.Sprintf("http://%v", vars["url"]))
	if err != nil {
		fmt.Println("ERROR in URLStatusHandler", err)
		return
	}

	status, err := utility.FetchURLStatus(parsedURL)
	if err != nil {
		fmt.Println("ERROR in URLStatusHandler", err)
		return
	}

	json.NewEncoder(w).Encode(status)

}
