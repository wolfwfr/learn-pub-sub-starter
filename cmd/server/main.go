package main

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
)

func main() {
	fmt.Println("Starting Peril server...")

	connStr := "amqp://guest:guest@localhost:5672/"
	conn, err := amqp.Dial(connStr)
	if err != nil {
		panic(fmt.Errorf("dialing amqp connection: %w", err))
	}
	defer conn.Close()
	fmt.Printf("connection was established successfully \n")

	ch, err := conn.Channel()
	if err != nil {
		panic(fmt.Errorf("creating channel: %w", err))
	}
	pubsub.PublishJSON(ch, routing.ExchangePerilDirect, routing.PauseKey, routing.PlayingState{
		IsPaused: true,
	})

	// _, _, err = pubsub.DeclareAndBind(conn, routing.ExchangePerilTopic, "game_logs", "game_logs.*", pubsub.Durable)
	// if err != nil {
	// 	panic(fmt.Errorf("declaring & binding game_logs queue: %w", err))
	// }

	pubsub.SubscribeGob(conn, routing.ExchangePerilTopic, routing.GameLogSlug, fmt.Sprintf("%s.*", routing.GameLogSlug), pubsub.Durable, handlerLogs())
	if err != nil {
		panic(fmt.Errorf("subscribing game_logs queue: %w", err))
	}

	gamelogic.PrintServerHelp()

inf:
	for {
		words := gamelogic.GetInput()
		if len(words) == 0 {
			continue
		}
		first := words[0]
		switch first {
		case routing.PauseKey:
			fmt.Printf("sending a pause message \n")
			pubsub.PublishJSON(ch, routing.ExchangePerilDirect, routing.PauseKey, routing.PlayingState{
				IsPaused: true,
			})
		case "resume":
			fmt.Printf("sending a resume message \n")
			pubsub.PublishJSON(ch, routing.ExchangePerilDirect, routing.PauseKey, routing.PlayingState{
				IsPaused: false,
			})
		case "quit":
			fmt.Printf("quit command received: exiting\n")
			break inf
		default:
			fmt.Printf("cannot parse %s\n", first)
		}

	}

	// sigC := make(chan os.Signal, 1)
	// signal.Notify(sigC, os.Interrupt, syscall.SIGTERM)
	// <-sigC
	//
	fmt.Printf("\nINTERRUPT or KILL signal received, shutting down server\n")
}
