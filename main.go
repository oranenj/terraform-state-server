package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"github.com/xo/dburl"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

var logstd = log.New(os.Stdout, "", 1)
var logerr = log.New(os.Stderr, "ERROR: ", 1)
var lockConflict = errors.New("Lock Conflict")

var db *sql.DB

func init_db() {
	x, err := dburl.Open(os.Args[1])
	if err != nil {
		logerr.Fatal("Failed to initialize DB driver:", err)
	}
	db = x

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS states (path TEXT PRIMARY KEY NOT NULL, value TEXT, lockid TEXT)")

	if err != nil {
		logerr.Fatal("Could not initialize database:", err)
	}
}

func db_get(path string) (string, error) {
	var value string
	row := db.QueryRow("SELECT value from states WHERE path = ?", path)
	err := row.Scan(&value)
	if err != nil {
		return "", err
	}
	return value, nil
}

func db_get_lock(path string) (string, error) {
	logstd.Println("Returning data for", path)
	var value string
	row := db.QueryRow("SELECT lockid from states WHERE path = $1", path)
	err := row.Scan(&value)
	if err != nil {
		return "", err
	}
	return value, nil
}

func db_lock(path string, lockid string) error {
	logstd.Println("Locking", path, lockid)
	_, err := db.Exec("INSERT INTO states values($1, $2, $3) ON CONFLICT(path) DO UPDATE set lockid = $3 WHERE (lockid IS NULL OR lockid = $3)", path, "", lockid)

	// Small window for a race, I guess, but should not matter in practice
	newlock, _ := db_get_lock(path)

	if newlock != lockid {
		logerr.Println("Locking error: Attempted to lock", path, "with", lockid, "but the current lock is:", newlock)
		return lockConflict
	}
	return err
}

func db_unlock(path string, lockid string) error {
	logstd.Println("Unlocking", path, lockid)
	r, err := db.Exec("UPDATE states SET lockid = NULL WHERE path = $1 AND lockid = $2", path, lockid)
	rows, row_error := r.RowsAffected()
	if rows == 0 {
		logerr.Println("Unable to unlock", path, lockid, rows, row_error)
		return lockConflict
	}
	return err
}

func db_put(path string, value string, lockid string) error {
	var l *string = &lockid
	if lockid == "" {
		l = nil
	}
	logstd.Println("Updating", path, "with lock", l)
	r, err := db.Exec("INSERT INTO states values($1, $2, $3) ON CONFLICT(path) DO UPDATE SET value = excluded.value WHERE lockid = excluded.lockid", path, value, l)
	if err != nil {
		logerr.Println("Error inserting data into database")
		return err
	}
	_, insert_error := r.LastInsertId()
	if insert_error != nil {
		logerr.Println("Lock conflict when putting data in:", path, l)
		return lockConflict
	}
	return nil
}

func get_body(r *http.Request) (string, error) {
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		data = []byte{}
	}
	return string(data), err
}

func get_id_from_body(body string) (string, error) {
	var data map[string]string
	err := json.Unmarshal([]byte(body), &data)
	str, exists := data["ID"]
	if exists == false {
		return "", errors.New("Lock ID not found")
	}
	return str, err
}

func request_handler(w http.ResponseWriter, r *http.Request) {
	body, body_err := get_body(r)
	switch r.Method {
	case "GET":
		value, err := db_get(r.URL.Path)
		if err == sql.ErrNoRows {
			logstd.Println("Not Found: ", r.URL.Path)
			http.Error(w, "Not Found", 404)
		} else if err != nil {
			http.Error(w, fmt.Sprintf("Database Error: %s", err), 503)
		} else {
			io.WriteString(w, value)
		}
		return
	case "DELETE":
		logstd.Println("DELETE")
		return
	case "POST":
		lockid := ""
		lockids, exists := r.URL.Query()["ID"]
		if exists {
			lockid = lockids[0]
		}
		if body_err != nil {
			http.Error(w, "Bad Request", 400)
		}
		err := db_put(r.URL.Path, body, lockid)
		if err != nil {
			http.Error(w, fmt.Sprintf("Database Error: %s", err), 503)
		}
		return
	case "LOCK":
		lockid, _ := get_id_from_body(body)
		err := db_lock(r.URL.Path, lockid)
		if err == lockConflict {
			http.Error(w, "Conflict", 409)
		} else if err != nil {
			http.Error(w, fmt.Sprintf("Database Error: %s", err), 503)
		}
		return
	case "UNLOCK":
		lockid, _ := get_id_from_body(body)
		err := db_unlock(r.URL.Path, lockid)
		if err == lockConflict {
			http.Error(w, "Conflict", 409)
		} else if err != nil {
			http.Error(w, fmt.Sprintf("Database Error: %s", err), 503)
		}
		return
	default:
		logerr.Println("Invalid request")
		http.Error(w, "Invalid Method", 405)
	}
}

func main() {
	http.HandleFunc("/", request_handler)
	bind := fmt.Sprintf("127.0.0.1:%s", os.Args[2])
	log.Println("Connecting to database on", os.Args[1])
	init_db()
	db.Ping()
	log.Println("Starting server on", bind)
	log.Fatal(http.ListenAndServe(bind, nil))
}
