package utility

import (
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
	Cooldown int
}

func InitializeDB() error {
	var err error
	db, err = buntdb.Open("data.db")
	if err != nil {
		return err
	}

	db.Update(func(tx *buntdb.Tx) error {
		tx.Set("config:blacklist", "www.reddit.com,reddit.com,www.facebook.com,facebook.com", nil)
		tx.Set("config:cooldown:news.ycombinator.com", "1", nil)
		return nil
	})

	return nil
}

func CloseDB() {
	db.Close()
}

func SetURLCooldown(url *url.URL) {
	var cooldownMinutes uint64
	var err error

	err = db.View(func(tx *buntdb.Tx) error {
		cooldownString, err := tx.Get(fmt.Sprintf("config:cooldown:%v", url.Hostname()))
		if err != nil {
			return err
		}

		cooldownMinutes, err = strconv.ParseUint(cooldownString, 10, 16)
		if err != nil {
			fmt.Printf("Invalid cooldown time [%v] for %v", cooldownString, url.Hostname())
			return err
		}

		return nil
	})

	if err == nil {
		db.Update(func(tx *buntdb.Tx) error {
			tx.Set(fmt.Sprintf("cooldown:%v", url.Hostname()), "true", &buntdb.SetOptions{
				Expires: true,
				TTL:     time.Duration(uint64(time.Minute) * cooldownMinutes),
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
		cooldownDuration, err = tx.TTL(fmt.Sprintf("cooldown:%v", url.Hostname()))
		return err
	})

	// Cooldown exists
	if err == nil {
		return &ProxyCopURLStatus{
			Cooldown: int(cooldownDuration / time.Second),
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
