package internal

import (
	"crypto/tls"
	"net/http"

	"github.com/gin-gonic/gin"
)

func RunAPI() {
	var err error

	var cert tls.Certificate

	Initialize()

	router := gin.Default()
	router.GET("/v1/primary", getPrimary)
	router.GET("/v1/primaries", getPrimaries)
	router.GET("/v1/standbys", getStandbys)
	router.GET("/v1/status/:id", getStatus)

	globalHandler.logger.Debugf("Running on %s", globalHandler.config.BindTo())

	if globalHandler.config.Ssl.Enabled() {
		globalHandler.logger.Debug("Running with SSL")

		cert, err = tls.X509KeyPair(globalHandler.config.Ssl.MustCertBytes(), globalHandler.config.Ssl.MustKeyBytes())
		if err != nil {
			globalHandler.logger.Fatal("Error parsing cert and key", err)
		}

		tlsConfig := tls.Config{
			MinVersion:   tls.VersionTLS12,
			Certificates: []tls.Certificate{cert},
		}
		server := http.Server{Addr: globalHandler.config.BindTo(), Handler: router, TLSConfig: &tlsConfig}
		err = server.ListenAndServeTLS("", "")
	} else {
		globalHandler.logger.Debug("Running without SSL")
		err = router.Run(globalHandler.config.BindTo())
	}

	if err != nil {
		globalHandler.logger.Panicf("Error running API: %s", err.Error())
	}
}

// getPrimary responds with the list of all albums as JSON.
func getPrimary(c *gin.Context) {
	primary := globalHandler.GetPrimaries()

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
	primaries := globalHandler.GetPrimaries()
	c.IndentedJSON(http.StatusOK, primaries)
}

// getStandbys responds with the list of all albums as JSON.
func getStandbys(c *gin.Context) {
	c.IndentedJSON(http.StatusOK, globalHandler.GetStandbys())
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
