package network

import (
	"net/http"

	"github.com/ElrondNetwork/elrond-proxy-go/api/shared"
	"github.com/ElrondNetwork/elrond-proxy-go/data"
	"github.com/ElrondNetwork/elrond-proxy-go/process"
	"github.com/gin-gonic/gin"
)

// Routes defines address related routes
func Routes(router *gin.RouterGroup) {
	router.GET("/status/:shard", GetNetworkStatusData)
	router.GET("/config", GetNetworkConfigData)
	router.GET("/economics", GetEconomicsData)
}

// GetNetworkStatusData will expose the node network metrics for the given shard
func GetNetworkStatusData(c *gin.Context) {
	ef, ok := c.MustGet("elrondProxyFacade").(FacadeHandler)
	if !ok {
		shared.RespondWithInvalidAppContext(c)
		return
	}

	shardIDUint, err := shared.FetchShardIDFromRequest(c)
	if err != nil {
		shared.RespondWith(c, http.StatusBadRequest, nil, process.ErrInvalidShardId.Error(), data.ReturnCodeRequestError)
		return
	}

	networkStatusResults, err := ef.GetNetworkStatusMetrics(shardIDUint)
	if err != nil {
		shared.RespondWith(c, http.StatusInternalServerError, nil, err.Error(), data.ReturnCodeInternalError)
		return
	}

	c.JSON(http.StatusOK, networkStatusResults)
}

// GetNetworkConfigData will expose the node network metrics for the given shard
func GetNetworkConfigData(c *gin.Context) {
	ef, ok := c.MustGet("elrondProxyFacade").(FacadeHandler)
	if !ok {
		shared.RespondWithInvalidAppContext(c)
		return
	}

	networkConfigResults, err := ef.GetNetworkConfigMetrics()
	if err != nil {
		shared.RespondWith(c, http.StatusInternalServerError, nil, err.Error(), data.ReturnCodeInternalError)
		return
	}

	c.JSON(http.StatusOK, networkConfigResults)
}

// GetEconomicsData will expose the economics data metrics from an observer (if any available) in json format
func GetEconomicsData(c *gin.Context) {
	ef, ok := c.MustGet("elrondProxyFacade").(FacadeHandler)
	if !ok {
		shared.RespondWithInvalidAppContext(c)
		return
	}

	economicsData, err := ef.GetEconomicsDataMetrics()
	if err != nil {
		shared.RespondWith(c, http.StatusInternalServerError, nil, err.Error(), data.ReturnCodeInternalError)
		return
	}

	c.JSON(http.StatusOK, economicsData)
}