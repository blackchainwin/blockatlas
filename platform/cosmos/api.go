package cosmos

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"github.com/trustwallet/blockatlas/coin"
	"github.com/trustwallet/blockatlas/models"
	"github.com/trustwallet/blockatlas/util"
)

var client = Client{
	HTTPClient: http.DefaultClient,
}

// Setup registers the Cosmos chain route
func Setup(router gin.IRouter) {
	router.Use(util.RequireConfig("cosmos.api"))
	router.Use(func(c *gin.Context) {
		client.BaseURL = viper.GetString("cosmos.api")
	})
	router.GET("/:address", getTransactions)
}

func getTransactions(c *gin.Context) {
	inputTxes, _ := client.GetAddrTxes(c.Param("address"), "inputs")
	outputTxes, _ := client.GetAddrTxes(c.Param("address"), "outputs")

	normalisedTxes := make([]models.Tx, 0)

	for _, inputTx := range inputTxes {
		normalisedInputTx := Normalize(&inputTx)
		normalisedTxes = append(normalisedTxes, normalisedInputTx)
	}
	for _, outputTx := range outputTxes {
		normalisedOutputTx := Normalize(&outputTx)
		normalisedTxes = append(normalisedTxes, normalisedOutputTx)
	}

	page := models.Response(normalisedTxes)
	page.Sort()
	c.JSON(http.StatusOK, &page)
}

// Normalize converts an Cosmos transaction into the generic model
func Normalize(srcTx *Tx) (tx models.Tx) {
	date, _ := time.Parse("2006-01-02T15:04:05Z", srcTx.Date)
	value, _ := util.DecimalToSatoshis(srcTx.Data.Contents.Message[0].Particulars.Amount[0].Quantity)
	block, _ := strconv.ParseUint(srcTx.Block, 10, 64)
	// Sometimes fees can be null objects (in the case of no fees e.g. F044F91441C460EDCD90E0063A65356676B7B20684D94C731CF4FAB204035B41)
	var fee string
	if len(srcTx.Data.Contents.Fee.FeeAmount) == 0 {
		fee = "0"
	} else {
		fee, _ = util.DecimalToSatoshis(srcTx.Data.Contents.Fee.FeeAmount[0].Quantity)
	}
	return models.Tx{
		ID:    srcTx.ID,
		Coin:  coin.ATOM,
		Date:  date.Unix(),
		From:  srcTx.Data.Contents.Message[0].Particulars.FromAddr, // This will need to be adjusted for multi-outputs, later
		To:    srcTx.Data.Contents.Message[0].Particulars.ToAddr,   // Likewise
		Fee:   models.Amount(fee),
		Block: block,
		Memo:  srcTx.Data.Contents.Memo,
		Meta: models.Transfer{
			Value: models.Amount(value),
		},
	}
}
