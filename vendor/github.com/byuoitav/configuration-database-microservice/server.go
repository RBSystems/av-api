package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/byuoitav/authmiddleware"
	"github.com/byuoitav/configuration-database-microservice/accessors"
	"github.com/byuoitav/configuration-database-microservice/handlers"
	"github.com/byuoitav/device-monitoring-microservice/statusinfrastructure"
	"github.com/jessemillar/health"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

func main() {
	database := os.Getenv("CONFIGURATION_DATABASE_USERNAME") + ":" + os.Getenv("CONFIGURATION_DATABASE_PASSWORD") + "@tcp(" + os.Getenv("CONFIGURATION_DATABASE_HOST") + ":" + os.Getenv("CONFIGURATION_DATABASE_PORT") + ")/" + os.Getenv("CONFIGURATION_DATABASE_NAME")

	// Constructs a new accessor group and connects it to the database
	accessorGroup := new(accessors.AccessorGroup)
	accessorGroup.Open(database)

	// Constructs a new controller group and gives it the accessor group
	handlerGroup := new(handlers.HandlerGroup)
	handlerGroup.Accessors = accessorGroup

	port := ":8006"
	router := echo.New()
	router.Pre(middleware.RemoveTrailingSlash())
	router.Use(middleware.CORS())

	// Use the `secure` routing group to require authentication
	secure := router.Group("", echo.WrapMiddleware(authmiddleware.Authenticate))

	router.GET("/health", echo.WrapHandler(http.HandlerFunc(health.Check)))
	router.GET("/mstatus", GetStatus)

	secure.GET("/buildings", handlerGroup.GetAllBuildings)
	secure.GET("/buildings/:id", handlerGroup.GetBuildingByID)
	secure.GET("/buildings/:shortname", handlerGroup.GetBuildingByShortname)
	secure.GET("/buildings/:building/rooms/:room", handlerGroup.GetRoomByBuildingAndName)
	secure.GET("/buildings/:building/rooms", handlerGroup.GetRoomsByBuilding)
	secure.GET("/buildings/:building/rooms/:room/devices", handlerGroup.GetDevicesByBuildingAndRoom)
	secure.GET("/buildings/:building/rooms/:room/devices/roles/:role", handlerGroup.GetDevicesByBuildingAndRoomAndRole)
	secure.GET("/buildings/:building/rooms/:room/devices/:device", handlerGroup.GetDeviceByBuildingAndRoomAndName)

	secure.PUT("/buildings/:building/rooms/:room/devices/:device/attributes/:attribute/:value", handlerGroup.PutDeviceAttributeByDeviceAndRoomAndBuilding)

	secure.GET("/rooms", handlerGroup.GetAllRooms)
	secure.GET("/rooms/designations", handlerGroup.GetAllRoomDesignations)
	secure.GET("/rooms/id/:id", handlerGroup.GetRoomByID)
	secure.GET("/rooms/buildings/:building", handlerGroup.GetRoomsByBuilding)

	//need to address these
	secure.GET("/rooms/:roomId/roles/:roleId", handlerGroup.GetDevicesByRoomIdAndRoleId)
	secure.GET("/rooms/:roomId/devices", handlerGroup.GetDevicesByRoomId)

	secure.GET("/devices/roles/:role/types/:type", handlerGroup.GetDevicesByRoleAndType)
	secure.GET("/deployment/devices/roles/:role/types/:type/:branch", handlerGroup.GetBranchDevicesByRoleAndType)

	secure.GET("/buildings/:building/rooms/:room/configuration", handlerGroup.GetConfigurationByRoomAndBuilding)
	secure.GET("/configurations/:configuration", handlerGroup.GetConfigurationByName)
	secure.GET("/configurations", handlerGroup.GetConfigurations)

	secure.GET("/devices/ports", handlerGroup.GetPorts)
	secure.GET("/devices/types", handlerGroup.GetDeviceTypes)
	secure.GET("/devices/classes", handlerGroup.GetDeviceClasses)
	secure.GET("/devices/endpoints", handlerGroup.GetEndpoints)
	secure.GET("/devices/commands", handlerGroup.GetAllCommands)
	secure.GET("/devices/powerstates", handlerGroup.GetPowerStates)
	secure.GET("/devices/microservices", handlerGroup.GetMicroservices)
	secure.GET("/devices/roledefinitions", handlerGroup.GetDeviceRoleDefs)
	secure.GET("/devices/roledefinitions/:id", handlerGroup.GetDeviceRoleDefsById)
	secure.GET("/devices/:id", handlerGroup.GetDeviceById)

	secure.GET("/classes/:class/ports", handlerGroup.GetPortsByDeviceType)

	secure.PUT("/devices/id/:deviceID/typeid", handlerGroup.SetDeviceTypeByID)
	secure.PUT("/devices/attribute", handlerGroup.SetDeviceAttribute)

	secure.POST("/buildings/:building", handlerGroup.AddBuilding)
	secure.POST("/buildings/:building/rooms/:room", handlerGroup.AddRoom)
	secure.POST("/buildings/:building/rooms/:room/devices/:device", handlerGroup.AddDevice)

	secure.POST("/devices/ports/:port", handlerGroup.AddPort)
	secure.POST("/devices/types/:devicetype", handlerGroup.AddDeviceType)
	secure.POST("/devices/endpoints/:endpoint", handlerGroup.AddEndpoint)
	secure.POST("/devices/commands/:command", handlerGroup.AddCommand)
	secure.POST("/devices/powerstates/:powerstate", handlerGroup.AddPowerState)
	secure.POST("/devices/microservices/:microservice", handlerGroup.AddMicroservice)
	secure.POST("/devices/roledefinitions/:deviceroledefinition", handlerGroup.AddDeviceRoleDef)

	//	secure.POST("/buildings/:building/rooms/:room/devices/:device/commands/:id", handlerGroup.AddDeviceCommand)
	//	secure.POST("/buildings/:building/rooms/:room/devices/:device/powerstates/:id", handlerGroup.AddDevicePowerState)
	//	secure.POST("/buildings/:building/rooms/:room/devices/:device/portconfiguration/:id", handlerGroup.AddPortConfiguration)
	//	secure.POST("/buildings/:building/rooms/:room/devices/:device/roles/:id", handlerGroup.AddDeviceRole)

	server := http.Server{
		Addr:           port,
		MaxHeaderBytes: 1024 * 10,
	}

	router.StartServer(&server)
}

func GetStatus(context echo.Context) error {
	var s statusinfrastructure.Status
	var err error

	s.Version, err = statusinfrastructure.GetVersion("version.txt")
	if err != nil {
		return context.JSON(http.StatusOK, "Failed to open version.txt")
	}

	// open new? database connection
	database := os.Getenv("CONFIGURATION_DATABASE_USERNAME") + ":" + os.Getenv("CONFIGURATION_DATABASE_PASSWORD") + "@tcp(" + os.Getenv("CONFIGURATION_DATABASE_HOST") + ":" + os.Getenv("CONFIGURATION_DATABASE_PORT") + ")/" + os.Getenv("CONFIGURATION_DATABASE_NAME")
	accessorGroup := new(accessors.AccessorGroup)
	accessorGroup.Open(database)

	vals, err := accessorGroup.GetAllBuildings()
	if len(vals) < 1 || err != nil {
		s.Status = statusinfrastructure.StatusDead
		s.StatusInfo = fmt.Sprintf("Unable to access database. Error: %s", err)
	} else {
		s.Status = statusinfrastructure.StatusOK
		s.StatusInfo = ""
	}

	accessorGroup.Database.Close()

	return context.JSON(http.StatusOK, s)
}
