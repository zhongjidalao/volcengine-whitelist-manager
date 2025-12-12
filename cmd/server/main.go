package main

import (
	"fmt"
	"log"

	"volcengine-updater/internal/config"
	"volcengine-updater/internal/service"
	"volcengine-updater/internal/web"

	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
)

func main() {
	// Initialize Database
	config.InitDB()

	// Initialize Scheduler
	c := cron.New()
	settings := config.GetSettings()
	
	var jobID cron.EntryID
	if settings.CheckInterval > 0 {
		id, err := c.AddFunc(fmt.Sprintf("@every %ds", settings.CheckInterval), service.CheckAndUpdate)
		if err != nil {
			log.Printf("Failed to schedule job: %v", err)
		} else {
			jobID = id
		}
	}
	c.Start()
	defer c.Stop()

	// Initialize Web Server
	r := gin.Default()
	r.LoadHTMLGlob("templates/*")
	// r.Static("/static", "./static")

	h := &web.Handler{
		Cron:  c,
		JobID: jobID,
	}
	h.RegisterRoutes(r)

	// Run
	log.Println("Starting server on :5000...")
	if err := r.Run(":5000"); err != nil {
		log.Fatal(err)
	}
}
