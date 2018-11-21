package utility

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/tidwall/buntdb"
)

var db *buntdb.DB

var ErrNoStatus error = errors.New("No Status")

type ProxyCopURLStatus struct {
	Blacklisted bool

	// Cooldown (TTL) in seconds
	Cooldown uint64
}

type ProxyCopURLConfig struct {
	// Cooldown (TTL) in seconds
	Cooldown uint64
}

func InitializeDB() error {
	var err error
	db, err = buntdb.Open("data.db")
	if err != nil {
		return err
	}

	db.Update(func(tx *buntdb.Tx) error {
		tx.Set("config:blacklist", "www.reddit.com,reddit.com,www.facebook.com,facebook.com", nil)
		tx.Set("url:news.ycombinator.com:config", `{"Cooldown": 1}`, nil)
		return nil
	})

	return nil
}

func CloseDB() {
	db.Close()
}

func GetURLConfig(url *url.URL) (*ProxyCopURLConfig, error) {

	config := &ProxyCopURLConfig{}

	err := db.View(func(tx *buntdb.Tx) error {
		configString, err := tx.Get(fmt.Sprintf("url:%v:config", url.Hostname()))
		if err != nil {
			return err
		}

		return json.Unmarshal([]byte(configString), config)
	})

	if err != nil {
		return config, err
	}

	return config, nil

}

func SetURLCooldown(url *url.URL) {
	var err error

	config, err := GetURLConfig(url)

	if err == nil {
		db.Update(func(tx *buntdb.Tx) error {
			tx.Set(fmt.Sprintf("url:%v:cooldown", url.Hostname()), "true", &buntdb.SetOptions{
				Expires: true,
				TTL:     time.Duration(uint64(time.Minute) * config.Cooldown),
			})
			return nil
		})
	}
}

func FetchURLStatus(url *url.URL) (*ProxyCopURLStatus, error) {
	var cooldownDuration time.Duration

	// Check to see if the current page has been visited too recenly (i.e. is on cooldown)
	err := db.View(func(tx *buntdb.Tx) error {
		var err error
		cooldownDuration, err = tx.TTL(fmt.Sprintf("url:%v:cooldown", url.Hostname()))
		return err
	})

	// Cooldown exists
	if err == nil {
		return &ProxyCopURLStatus{
			Cooldown: uint64(cooldownDuration / time.Second),
		}, nil
	}

	var forbiddenURLs []string

	// Collect forbidden URLs that may never be visited
	db.View(func(tx *buntdb.Tx) error {
		blacklistString, _ := tx.Get("config:blacklist")
		forbiddenURLs = strings.Split(blacklistString, ",")
		return nil
	})

	for _, val := range forbiddenURLs {
		if url.Hostname() == val {
			return &ProxyCopURLStatus{
				Blacklisted: true,
			}, nil
		}
	}

	return nil, ErrNoStatus
}

func IncrementKey(key string) {
	db.Update(func(tx *buntdb.Tx) error {

		count := uint64(0)

		domainCountStr, err := tx.Get(key)

		if err != nil {
			count, err = strconv.ParseUint(domainCountStr, 10, 64)
			if err != nil {
				fmt.Printf("Invalid domain count [%v] for %v", domainCountStr, key)
				return err
			}
		}

		count++

		tx.Set(key, fmt.Sprintf("%v", count), nil)

		return nil
	})
}

func RecordURL(url *url.URL) {
	IncrementKey(fmt.Sprintf("url:%v:stats:count", url.Hostname()))
}
