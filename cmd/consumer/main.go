package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

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
	foverever := make(chan bool)    // outro channel
	go rabbitmq.Consume(ch, out)    // thread 2

	qtdWorkers := 150
	for i := 1; i <= qtdWorkers; i++ {
		go worker(out, &uc, i) // Criando mais threads
	}
	<-foverever // nesse momento a aplicação fica travada pois em nenhum momento esse canal recebeu alguma msg
}

func worker(deliveryMessage <-chan amqp.Delivery, uc *usecase.CalculateFinalPriceUseCase, workerId int) {

	for msg := range deliveryMessage {
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
		fmt.Printf("Worker %d has processed order %s\n", workerId, outputDTO.ID) // thread 1 (pertence a thread do main)
		time.Sleep(1 * time.Second)
	}
}
