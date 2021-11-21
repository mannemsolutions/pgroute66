package internal

import (
	"crypto/tls"
	"github.com/gin-gonic/gin"
	"net/http"
)

func must(fn func() (interface{}, error)) interface{} {
	v, err := fn()
	if err != nil {
		log.Fatal(err)
	}
	return v
}

func RunAPI() {
	var err error
	router := gin.Default()
	router.GET("/v1/primary", getPrimary)
	router.GET("/v1/primaries", getPrimaries)
	router.GET("/v1/standbys", getStandbys)
	router.GET("/v1/status/:id", getStatus)

	if config.Ssl.Enabled() {
		cert, err := tls.X509KeyPair(config.Ssl.MustCertBytes(), config.Ssl.MustKeyBytes())
		if err != nil {
			log.Fatal("Error parsing cert and key", err)
		}
		tlsConfig := tls.Config{Certificates: []tls.Certificate{cert}}
		server := http.Server{Addr: config.BindTo(), Handler: router, TLSConfig: &tlsConfig}
		err = server.ListenAndServeTLS("", "")
	} else {
		err = router.Run(config.BindTo())
	}
	if err != nil {
		log.Panicf("Error running API: %s", err.Error())
	}
}

// getPrimary responds with the list of all albums as JSON.
func getPrimary(c *gin.Context) {
	primary := handler.GetPrimaries()

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
	primaries := handler.GetPrimaries()
	c.IndentedJSON(http.StatusOK, primaries)
}
// getStandbys responds with the list of all albums as JSON.
func getStandbys(c *gin.Context) {
	c.IndentedJSON(http.StatusOK, handler.GetStandbys())
}

func getStatus(c *gin.Context) {
	id := c.Param("id")
	status := handler.GetNodeStatus(id)
	switch status {
	case "primary", "standby":
		c.IndentedJSON(http.StatusOK, status)
	case "invalid":
		c.IndentedJSON(http.StatusNotFound, status)
	case "unavailable":
		c.IndentedJSON(http.StatusUnprocessableEntity, status)
	}
}