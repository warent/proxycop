package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/elazarl/goproxy"
	"github.com/gorilla/mux"
	"github.com/tidwall/buntdb"
	"github.com/warent/proxycop/apiroutes"
)

var DB *buntdb.DB
var sessions map[int64]*string
var sessionsWriteMutex sync.Mutex

func writeSessionResult(session int64, result string) {
	sessionsWriteMutex.Lock()
	sessions[session] = &result
	sessionsWriteMutex.Unlock()
}

func filterSession(r *http.Request, ctx *goproxy.ProxyCtx) bool {
	if val, ok := sessions[ctx.Session]; ok {
		if val != nil {
			return true
		}
		return false
	}
	return isRequestForbidden(r, ctx)
}

func isRequestForbidden(r *http.Request, ctx *goproxy.ProxyCtx) bool {

	var cooldownDuration time.Duration

	// Check to see if the current page has been visited too recenly (i.e. is on cooldown)
	err := DB.View(func(tx *buntdb.Tx) error {
		var err error
		cooldownDuration, err = tx.TTL(fmt.Sprintf("cooldown:%v", r.URL.Hostname()))
		return err
	})

	// Cooldown exists
	if err == nil {
		writeSessionResult(ctx.Session,
			fmt.Sprintf("Your cooldown for this page is still pending. Please wait %v", cooldownDuration))
		return true
	}

	var forbiddenURLs []string

	// Collect forbidden URLs that may never be visited
	DB.View(func(tx *buntdb.Tx) error {
		blacklistString, _ := tx.Get("config:blacklist")
		forbiddenURLs = strings.Split(blacklistString, ",")
		return nil
	})

	for _, val := range forbiddenURLs {
		if r.URL.Hostname() == val {
			writeSessionResult(ctx.Session, "This page is blacklisted from being visited.")
			return true
		}
	}

	var cooldownMinutes uint64

	err = DB.View(func(tx *buntdb.Tx) error {
		cooldownString, err := tx.Get(fmt.Sprintf("config:cooldown:%v", r.URL.Hostname()))
		if err != nil {
			return err
		}

		cooldownMinutes, err = strconv.ParseUint(cooldownString, 10, 16)
		if err != nil {
			fmt.Printf("Invalid cooldown time [%v] for %v", cooldownString, r.URL.Hostname())
			return err
		}

		return nil
	})

	if err == nil {
		DB.Update(func(tx *buntdb.Tx) error {
			tx.Set(fmt.Sprintf("cooldown:%v", r.URL.Hostname()), "true", &buntdb.SetOptions{
				Expires: true,
				TTL:     time.Duration(uint64(time.Minute) * cooldownMinutes),
			})
			return nil
		})
	}

	return false
}

func startProxy() {

	var err error

	sessions = map[int64]*string{}

	DB, err = buntdb.Open("data.db")
	if err != nil {
		log.Fatal(err)
	}
	defer DB.Close()

	proxy := goproxy.NewProxyHttpServer()

	DB.Update(func(tx *buntdb.Tx) error {
		tx.Set("config:blacklist", "www.reddit.com,reddit.com,www.facebook.com,facebook.com", nil)
		tx.Set("config:cooldown:news.ycombinator.com", "1", nil)
		return nil
	})

	filterForbidden := func() goproxy.ReqConditionFunc {
		return filterSession
	}

	proxy.OnRequest(filterForbidden()).HandleConnect(goproxy.AlwaysMitm)
	proxy.OnRequest(filterForbidden()).DoFunc(func(r *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
		return r, goproxy.TextResponse(r, *sessions[ctx.Session])
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
