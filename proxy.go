package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"

	"github.com/elazarl/goproxy"
	"github.com/gorilla/mux"
	"github.com/warent/proxycop/apiroutes"
	"github.com/warent/proxycop/utility"
)

var sessions map[int64]bool
var sessionsWriteMutex sync.Mutex

func writeSessionResult(session int64, result bool) {
	sessionsWriteMutex.Lock()
	sessions[session] = result
	sessionsWriteMutex.Unlock()
}

func filterSession(r *http.Request, ctx *goproxy.ProxyCtx) bool {
	if val, ok := sessions[ctx.Session]; ok {
		return val
	}
	sessions[ctx.Session] = isRequestForbidden(r, ctx)
	return sessions[ctx.Session]
}

func isRequestForbidden(r *http.Request, ctx *goproxy.ProxyCtx) bool {

	_, err := utility.FetchURLStatus(r.URL)

	if err != nil {
		return true
	}

	utility.SetURLCooldown(r.URL)

	return false
}

func startProxy() {

	sessions = map[int64]bool{}

	proxy := goproxy.NewProxyHttpServer()

	filterForbidden := func() goproxy.ReqConditionFunc {
		return filterSession
	}

	proxy.OnRequest(filterForbidden()).HandleConnect(goproxy.AlwaysMitm)
	proxy.OnRequest(filterForbidden()).DoFunc(func(r *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
		w := httptest.NewRecorder()
		http.Redirect(w, r, fmt.Sprintf("http://proxy.cop/status/url/%v", r.URL.Hostname()), http.StatusSeeOther)
		return r, w.Result()
	})

	proxy.OnRequest(goproxy.DstHostIs("proxy.cop")).DoFunc(func(r *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {

		w := httptest.NewRecorder()

		router := mux.NewRouter()

		router.HandleFunc("/api/config", apiroutes.ConfigHandler)

		if strings.HasPrefix(r.URL.Path, "/static") {
			fs := http.FileServer(http.Dir("frontend/dist"))
			fs.ServeHTTP(w, r)
		} else if strings.HasPrefix(r.URL.Path, "/api") {
			router.ServeHTTP(w, r)
		} else {
			http.ServeFile(w, r, "frontend/dist/index.html")
		}

		return r, w.Result()

	})

	log.Fatal(http.ListenAndServe(":8080", proxy))
}
