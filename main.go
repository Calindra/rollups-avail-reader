package main

import (
	"fmt"
	"time"

	"github.com/calindra/rollups-avail-reader/pkg/paioavail"
	"github.com/calindra/rollups-base-reader/pkg/transaction"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

const timeout = 30 * time.Second

func main() {
	e := echo.New()
	e.Use(middleware.CORS())
	e.Use(middleware.Recover())
	e.Use(middleware.TimeoutWithConfig(middleware.TimeoutConfig{
		ErrorMessage: "Request timed out",
		Timeout:      timeout,
	}))

	paio := paioavail.NewPaioSender2Server("")
	sender := transaction.TransactionAPI{
		ClientSender: paio,
	}
	transaction.Register(e, &sender)

	fmt.Println("Hello, from Avail Server!")

	e.Logger.Fatal(e.Start(":8082"))
}
