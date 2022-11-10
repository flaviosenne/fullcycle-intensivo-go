package main

import (
	"database/sql"
	"encoding/json"

	"github.com/flaviosenne/go-intensivo/internal/order/infra/database"
	"github.com/flaviosenne/go-intensivo/internal/order/usecase"
	"github.com/flaviosenne/go-intensivo/pkg/rabbitmq"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	db, err := sql.Open("sqlite3", "./orders.db")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	repository := database.NewOrderRepository(db)
	uc := usecase.CalculateFinalPriceUseCase{OrderRepository: repository}

	ch, err := rabbitmq.OpenChannel()
	if err != nil {
		panic(err)
	}
	defer ch.Close()
	out := make(chan amqp.Delivery) // chanel do go
	go rabbitmq.Consume(ch, out)    // thread 2

	for msg := range out {
		var inputDTO usecase.OrderInputDTO

		err := json.Unmarshal(msg.Body, &inputDTO)
		if err != nil {
			panic(err)
		}
		outputDTO, err := uc.Execute(inputDTO)
		if err != nil {
			panic(err)
		}
		msg.Ack(false)
		println(outputDTO) // thread 1 (pertence a thread do main)
	}
}
