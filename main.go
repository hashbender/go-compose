package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

func main() {
	var port = ""
	if os.Getenv("PORT") != "" {
		port = ":" + os.Getenv("PORT")
	} else {
		port = ":5000"
	}

	log.Println("Opening db: " + os.Getenv("DATABASE_URL"))
	var db *sql.DB
	var errd error
	for i := 0; i <= 10; i++ {
		db, errd = sql.Open("postgres", os.Getenv("DATABASE_URL"))
		if errd != nil {
			log.Println("Error opening database, retrying", errd)
		}
		if errd = db.Ping(); errd != nil {
			log.Println("Error opening database, retrying", errd)
		} else {
			log.Println("Got a database connection")
			break
		}
		time.Sleep(time.Second * 2)
	}

	if db == nil {
		panic(errd)
	}

	defer db.Close()
	pool := initRedis(db)
	defer pool.Close()

	db.SetMaxIdleConns(15)
	db.SetMaxOpenConns(20)

	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Hello world!")
		fmt.Fprint(w, "Hello world")
	})
	log.Println("Listening on port: " + port)
	http.ListenAndServe(port, router)
}

func initRedis(db *sql.DB) *redis.Pool {
	var pool *redis.Pool

	redisUrl := os.Getenv("REDIS_URL")
	if redisUrl != "" {
		for i := 0; i <= 10; i++ {
			log.Println("Opening redis: " + redisUrl)
			u, errd := url.Parse(redisUrl)
			if errd != nil {
				log.Print("Error parsing redis URL:", errd)
			}
			var password string
			if u.User == nil {
				log.Print("No user defined")
				password = ""
			} else {
				password, _ = u.User.Password()
			}

			subscribeConnection, errd := redis.Dial("tcp", u.Host)
			if errd != nil {
				log.Print("Error dialing redis host", errd)
			}

			if password != "" {
				_, errd = subscribeConnection.Do("AUTH", password)
				if errd != nil {
					log.Print("Error passing auth to redis", errd)
				}
			}
			pool = newPool(u.Host, password)
			if pool != nil {
				break
			} else {
				time.Sleep(time.Second * 5)
			}
		}
	} else {
		log.Println("Opening default redis connection")
		pool = newPool(":6379", "")
	}

	log.Println("Subscribing")
	subscribeConnection := pool.Get()

	subCon := redis.PubSubConn{Conn: subscribeConnection}
	subCon.Subscribe("ENGINE_UPDATE")
	go func(pool *redis.Pool) {
		for {
			switch message := subCon.Receive().(type) {
			case redis.Message:
				if message.Channel == "ENGINE_UPDATE" {
					orgID, err := strconv.Atoi(string(message.Data))
					if err != nil {
						log.Printf("ERROR getting an engine update for org: %d", orgID)
						continue
					}
					if err != nil {
						log.Printf("Error getting roles for org: %d", orgID)
						continue
					}
					log.Printf("Got an update!")
				}
			case redis.Subscription:
				log.Printf("Subscribed: %s", message.Channel)
			case error:
				log.Println("PubSubConn Error " + message.Error())
			}
		}
	}(pool)
	log.Println("Finished initializing redis")
	return pool
}

func newPool(server string, password string) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     30,
		MaxActive:   220,
		IdleTimeout: 1 * time.Minute,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", server)
			if err != nil {
				return nil, err
			}

			if password == "" {
				return c, err
			}

			if _, err := c.Do("AUTH", password); err != nil {
				c.Close()
				return nil, err
			}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			if time.Since(t) < 10*time.Second {
				return nil
			}
			_, err := c.Do("PING")
			return err
		},
	}
}
