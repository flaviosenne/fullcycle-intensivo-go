package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
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
	go rabbitmq.Consume(ch, out)    // thread 2

	qtdWorkers := 100
	for i := 1; i <= qtdWorkers; i++ {
		go worker(out, &uc, i) // Criando mais threads
	}

	http.HandleFunc("/total", func(w http.ResponseWriter, r *http.Request) {
		getTotalUC := usecase.GetTotalUseCase{OrderRepository: repository}
		total, err := getTotalUC.Execute()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
		}
		json.NewEncoder(w).Encode(total)
	})

	http.ListenAndServe(":8080", nil) // http cria mais uma thread
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
