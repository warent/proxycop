package apiroutes

import (
	"fmt"
	"net/http"
)

func ConfigHandler(w http.ResponseWriter, r *http.Request) {

	fmt.Fprintln(w, "Hello api")

}
