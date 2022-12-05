package internal

import (
	"crypto/tls"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func RunAPI() {
	var err error

	var cert tls.Certificate

	Initialize()

	if !globalHandler.config.Debug() {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()
	router.GET("/v1/primary", getPrimary)
	router.GET("/v1/primaries", getPrimaries)
	router.GET("/v1/standbys", getStandbys)
	router.GET("/v1/:id/status", getStatus)
	router.GET("/v1/:id/availability", getAvailability)

	log.Debugf("Running on %s", globalHandler.config.BindTo())

	if globalHandler.config.Ssl.Enabled() {
		log.Debug("Running with SSL")

		cert, err = tls.X509KeyPair(globalHandler.config.Ssl.MustCertBytes(), globalHandler.config.Ssl.MustKeyBytes())
		if err != nil {
			log.Fatal("Error parsing cert and key", err)
		}

		tlsConfig := tls.Config{
			MinVersion:   tls.VersionTLS12,
			Certificates: []tls.Certificate{cert},
		}
		server := http.Server{Addr: globalHandler.config.BindTo(), Handler: router, TLSConfig: &tlsConfig}
		err = server.ListenAndServeTLS("", "")
	} else {
		log.Debug("Running without SSL")
		err = router.Run(globalHandler.config.BindTo())
	}

	if err != nil {
		log.Panicf("Error running API: %s", err.Error())
	}
}

func getPrimary(c *gin.Context) {
	primary := globalHandler.GetPrimaries(c.DefaultQuery("group", "all"))
	switch len(primary) {
	case 0:
		c.IndentedJSON(http.StatusNotFound, "")
	case 1:
		c.IndentedJSON(http.StatusOK, primary[0])
	default:
		c.IndentedJSON(http.StatusConflict, "")
	}

}

// getPrimaries responds with the list of all albums as JSON.
func getPrimaries(c *gin.Context) {
	primaries := globalHandler.GetPrimaries(c.DefaultQuery("group", "all"))
	c.IndentedJSON(http.StatusOK, primaries)
}

// getStandbys responds with the list of all albums as JSON.
func getStandbys(c *gin.Context) {
	c.IndentedJSON(http.StatusOK, globalHandler.GetStandbys(c.DefaultQuery("group", "all")))
}

func getStatus(c *gin.Context) {
	id := c.Param("id")

	status := globalHandler.GetNodeStatus(id)
	switch status {
	case "primary", "standby":
		c.IndentedJSON(http.StatusOK, status)
	case "invalid":
		c.IndentedJSON(http.StatusNotFound, status)
	case "unavailable":
		c.IndentedJSON(http.StatusUnprocessableEntity, status)
	}
}
func getAvailability(c *gin.Context) {
	id := c.Param("id")

	var limit float64
	var err error
	if value := c.DefaultQuery("limit", "10"); value == "" {
		limit = -1
	} else if limit, err = strconv.ParseFloat(value, 32); err != nil {
		log.Errorf("invalid value for limit (%s is not an int32)", value)
	}

	status := globalHandler.GetNodeAvailability(id, limit)
	switch status {
	case "ok":
		c.IndentedJSON(http.StatusOK, status)
	case "exceeded":
		c.IndentedJSON(http.StatusRequestTimeout, status)
	default:

		c.IndentedJSON(http.StatusExpectationFailed, status)
	}
}
