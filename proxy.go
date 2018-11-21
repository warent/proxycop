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
	"github.com/rs/cors"
	"github.com/warent/proxycop/apiroutes"
	"github.com/warent/proxycop/utility"
)

var sessions map[int64]bool
var sessionsWriteMutex sync.Mutex

func safeWriteSessionResult(session int64, result bool) {
	sessionsWriteMutex.Lock()
	sessions[session] = result
	sessionsWriteMutex.Unlock()
}

func filterSession(r *http.Request, ctx *goproxy.ProxyCtx) bool {
	if val, ok := sessions[ctx.Session]; ok {
		return val
	}
	safeWriteSessionResult(ctx.Session, isRequestForbidden(r, ctx))
	return sessions[ctx.Session]
}

func isRequestForbidden(r *http.Request, ctx *goproxy.ProxyCtx) bool {

	status, err := utility.FetchURLStatus(r.URL)

	if err != nil && err != utility.ErrNoStatus {
		fmt.Println("Error: ", err)
	}

	if status != nil {
		return true
	}

	utility.SetURLCooldown(r.URL)

	return false
}

func startProxy() {

	c := cors.New(cors.Options{
		AllowedOrigins: []string{"http://proxy.cop", "http://localhost:8081"},
	})

	sessions = map[int64]bool{}

	proxy := goproxy.NewProxyHttpServer()

	filterForbidden := func() goproxy.ReqConditionFunc {
		return filterSession
	}

	proxy.OnRequest(filterForbidden()).HandleConnect(goproxy.AlwaysMitm)
	proxy.OnRequest(filterForbidden()).DoFunc(func(r *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
		w := httptest.NewRecorder()
		http.Redirect(w, r, fmt.Sprintf("http://proxy.cop/url/%v/status", r.URL.Hostname()), http.StatusSeeOther)
		return r, w.Result()
	})

	proxy.OnRequest(goproxy.DstHostIs("proxy.cop")).DoFunc(func(r *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {

		w := httptest.NewRecorder()

		router := mux.NewRouter()

		router.HandleFunc("/api/config", apiroutes.ConfigHandler)
		router.HandleFunc("/api/url/{url}/status", apiroutes.URLStatusHandler)

		if strings.HasPrefix(r.URL.Path, "/static") {
			http.FileServer(http.Dir("frontend/dist")).ServeHTTP(w, r)
		} else if strings.HasPrefix(r.URL.Path, "/api") {
			c.ServeHTTP(w, r, router.ServeHTTP)
		} else {
			http.ServeFile(w, r, "frontend/dist/index.html")
		}

		return r, w.Result()

	})

	log.Fatal(http.ListenAndServe(":8080", proxy))
}
