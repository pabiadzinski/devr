package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"time"
)

func log(level, msg string, fields map[string]any) string {
	entry := map[string]any{
		"time":  time.Now().Format("15:04:05.000"),
		"level": level,
		"msg":   msg,
	}

	for k, v := range fields {
		entry[k] = v
	}

	data, _ := json.Marshal(entry)
	line := string(data)
	fmt.Fprintln(os.Stdout, line)

	return line
}

func main() {
	log("info", "starting server", map[string]any{"port": 8080, "env": "development"})
	log("info", "connected to database", map[string]any{"host": "localhost:5432", "db": "myapp"})
	log("debug", "loading config", map[string]any{"file": "config.yaml"})

	endpoints := []string{"/api/users", "/api/orders", "/api/products", "/health"}
	methods := []string{"GET", "POST", "PUT", "DELETE"}
	statuses := []int{200, 200, 200, 200, 201, 204, 400, 404, 500}

	for {
		time.Sleep(time.Duration(200+rand.Intn(800)) * time.Millisecond)

		endpoint := endpoints[rand.Intn(len(endpoints))]
		method := methods[rand.Intn(len(methods))]
		status := statuses[rand.Intn(len(statuses))]
		dur := rand.Intn(500)

		fields := map[string]any{
			"method":   method,
			"path":     endpoint,
			"status":   status,
			"duration": fmt.Sprintf("%dms", dur),
		}

		switch {
		case status >= 500:
			log("error", "internal server error", fields)
		case status >= 400:
			log("warn", "client error", fields)
		case endpoint == "/health":
			log("debug", "health check", fields)
		default:
			log("info", "request handled", fields)
		}
	}
}
