package health

import (
	"log"
	"net/http"

	"github.com/byuoitav/av-api/base"
	"github.com/byuoitav/common/db"
	"github.com/byuoitav/common/health"
	"github.com/labstack/echo"
)

//const version = "0.9.1"

func GetHealth() map[string]string {

	log.Printf("[HealthCheck] Checking microservice health: ")

	healthReport := make(map[string]string)

	healthReport["Initialized"] = "ok"
	healthReport["Web Server Status"] = "ok"
	//	healthReport["Version"] = version

	vals, err := db.GetDB().GetBuildings()

	if len(vals) < 1 || err != nil {
		healthReport["Configuration Database Microservice Connectivity"] = "ERROR"
	} else {
		healthReport["Configuration Database Microservice Connectivity"] = "ok"
	}

	log.Printf("[HealthCheck] Done. Report:")
	for k, v := range healthReport {
		log.Printf("%v: %v", k, v)
	}
	log.Printf("[HealthCheck] End.")

	return healthReport
}

func Status(context echo.Context) error {
	report := GetHealth()

	return context.JSON(http.StatusOK, report)
}

func StartupCheckAndReport() {
	health.SendSuccessfulStartup(GetHealth, "AV-API", base.PublishHealth)
}
