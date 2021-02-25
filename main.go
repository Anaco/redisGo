package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/anaco/redisGo/db"
	"github.com/gin-gonic/gin"
	cors "github.com/rs/cors/wrapper/gin"
)

var (
	//ListenAddr where we are serving the redis requests from
	ListenAddr = "localhost:8080"
	//RedisAddr the redis instance
	RedisAddr = "localhost:6379"
)

func main() {
	ddbClient, err := db.NewDynamoDbClient("GemsAccountTable")
	if err != nil {
		log.Fatalf("Failed to create Dynamo Document Client, %s", err.Error())
	}
	database, err := db.NewDatabase(RedisAddr, ddbClient)
	if err != nil {
		log.Fatalf("Failed to connect to redis: %s", err.Error())
	}
	router := initRouter(database)
	router.Run(ListenAddr)
}

//initRouter register routes with the app
func initRouter(database *db.Database) *gin.Engine {
	r := gin.Default()
	r.Use(cors.Default())
	r.POST("/license/claim", func(c *gin.Context) {
		var licenseJSON db.License
		if err := c.ShouldBindJSON(&licenseJSON); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		_, err := database.CreateReservation(&licenseJSON)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusAccepted, gin.H{"reservation": licenseJSON})
	})
	//SetReserveLicense Endpoint
	// r.POST("/license/reserve", func(c *gin.Context) {
	// 	var licenseJSON db.License
	// 	if err := c.ShouldBindJSON(&licenseJSON); err != nil {
	// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	// 		return
	// 	}
	// 	err := database.ReserveLicense(&licenseJSON)
	// 	if err != nil {
	// 		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	// 		return
	// 	}
	// 	c.JSON(http.StatusOK, gin.H{"license": licenseJSON})
	// })
	r.GET("/license/fetchall", func(c *gin.Context) {
		appID := c.Query("appID")
		accountID := c.Query("accountID")
		record, err := database.FetchAccountReservations(accountID, appID)
		if err != nil {
			if err == db.ErrNil {
				c.JSON(http.StatusNotFound, gin.H{"error": "No records found for " + accountID + appID})
				return
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}
		c.JSON(http.StatusOK, gin.H{"licenses": record})
	})
	r.GET("/license/user/getLicense", func(c *gin.Context) {
		appID := c.Query("appID")
		accountID := c.Query("accountID")
		userID := c.Query("userID")
		license, err := database.FetchUserReservation(userID, appID, accountID)
		fmt.Printf("License:: %v", err)
		if err != nil {
			if err == db.ErrNil {
				c.JSON(http.StatusNotFound, gin.H{"error": "No records found for " + userID + " " + appID})
				return
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}
		c.JSON(http.StatusOK, gin.H{"license": license})
	})
	r.POST("/license/user/freeLicense", func(c *gin.Context){
		appID := c.Query("appID")
		accountID := c.Query("accountID")
		userID := c.Query("userID")
		err := database.ReturnUserLicense(userID, appID, accountID)
		if err != nil {
			if err == db.ErrNil{
				c.JSON(http.StatusNotFound, gin.H{"error":"No record found to revoke"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not revoke license"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "License revoked successfully"})

	})
	//FetchReserveLicense Endpoint
	// r.GET("/license/fetch", func(c *gin.Context) {
	// 	userID := c.Query("userID")
	// 	appID := c.Query("appID")
	// 	record, err := database.FetchReservedLicense(userID + appID)
	// 	if err != nil {
	// 		if err == db.ErrNil {
	// 			c.JSON(http.StatusNotFound, gin.H{"error": "No records found for " + userID + appID})
	// 			return
	// 		} else {
	// 			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	// 			return
	// 		}
	// 	}
	// 	c.JSON(http.StatusOK, gin.H{"record": record})
	// })
	return r
}
