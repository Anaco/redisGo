package main

import (
	"log"
	"net/http"

	"github.com/anaco/redisGo/db"
	"github.com/gin-gonic/gin"
)

var (
	//ListenAddr where we are serving the redis requests from
	ListenAddr = "localhost:8080"
	//RedisAddr the redis instance
	RedisAddr = "localhost:6379"
)

func main() {
	database, err := db.NewDatabase(RedisAddr)
	if err != nil {
		log.Fatalf("Failed to connect to redis: %s", err.Error())
	}
	router := initRouter(database)
	router.Run(ListenAddr)
}

//initRouter register routes with the app
func initRouter(database *db.Database) *gin.Engine {
	r := gin.Default()
	r.POST("/license/claim", func(c *gin.Context) {
		var licenseJSON db.License
		if err := c.ShouldBindJSON(&licenseJSON); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		err := database.CreateReservation(&licenseJSON)
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
		if err != nil {
			if err == db.ErrNil {
				c.JSON(http.StatusNotFound, gin.H{"error": "No records found for " + userID + " " + appID})
				return
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			}
		}
		c.JSON(http.StatusOK, gin.H{"license": license})
		return
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
