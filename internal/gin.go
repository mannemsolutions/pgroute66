package internal

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func RunAPI() {
	router := gin.Default()
	router.GET("/v1/primary", getPrimary)
	router.GET("/v1/primaries", getPrimaries)
	router.GET("/v1/standbys", getStandbys)
	router.GET("/v1/status/:id", getStatus)

	router.Run(config.BindTo())
}

// getPrimary responds with the list of all albums as JSON.
func getPrimary(c *gin.Context) {
	primary := handler.GetPrimaries()
	if len(primary) == 1 {
		c.IndentedJSON(http.StatusOK, primary)
	}
	c.IndentedJSON(http.StatusConflict, []string{})
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
		c.IndentedJSON(http.StatusOK, []string{status})
	case "invalid":
		c.IndentedJSON(http.StatusNotFound, []string{status})
	case "unavailable":
		c.IndentedJSON(http.StatusUnprocessableEntity, []string{status})
	}
}