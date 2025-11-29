package main

import (
	"fmt"
	"strconv"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
)

func main() {
	fmt.Println("Starting Peril client...")

	connStr := "amqp://guest:guest@localhost:5672/"
	conn, err := amqp.Dial(connStr)
	if err != nil {
		panic(fmt.Errorf("dialing amqp connection: %w", err))
	}
	defer conn.Close()
	fmt.Printf("connection was established successfully \n")

	username, err := gamelogic.ClientWelcome()
	if err != nil {
		panic(fmt.Errorf("calling clientWelcome: %w", err))
	}

	ch, _, err := pubsub.DeclareAndBind(conn, routing.ExchangePerilDirect, fmt.Sprintf("%s.%s", routing.PauseKey, username), routing.PauseKey, pubsub.Transient)
	if err != nil {
		panic(fmt.Errorf("declaring & binding queue: %w", err))
	}

	gamestate := gamelogic.NewGameState(username)

	// sub to moves
	err = pubsub.SubscribeJSON(conn, routing.ExchangePerilTopic, fmt.Sprintf("%s.%s", "army_moves", username), "army_moves.*", pubsub.Transient, handlerArmyMove(gamestate, ch))
	if err != nil {
		panic(fmt.Errorf("subscribing army-move-handler: %w", err))
	}

	// sub to war
	err = pubsub.SubscribeJSON(conn, routing.ExchangePerilTopic, "war", fmt.Sprintf("%s.%s", routing.WarRecognitionsPrefix, username), pubsub.Durable, handlerWar(gamestate, ch))
	if err != nil {
		panic(fmt.Errorf("subscribing war-handler: %w", err))
	}

	// sub to pause
	err = pubsub.SubscribeJSON(conn, routing.ExchangePerilDirect, fmt.Sprintf("%s.%s", routing.PauseKey, username), routing.PauseKey, pubsub.Transient, handlerPause(gamestate))
	if err != nil {
		panic(fmt.Errorf("subscribing pause-handler: %w", err))
	}

inf:
	for {
		words := gamelogic.GetInput()
		if len(words) == 0 {
			continue
		}
		first := words[0]
		switch first {
		case "spawn":
			if err := gamestate.CommandSpawn(words); err != nil {
				fmt.Printf("%s\n", err.Error())
				continue
			}
		case "move":
			mv, err := gamestate.CommandMove(words)
			if err != nil {
				fmt.Printf("%s\n", err.Error())
				continue
			}

			err = pubsub.PublishJSON(ch, routing.ExchangePerilTopic, fmt.Sprintf("army_moves.%s", username), mv)
			if err != nil {
				fmt.Printf("%s\n", err.Error())
				continue
			}
			fmt.Printf("successful move\n")
		case "status":
			gamestate.CommandStatus()
		case "help":
			gamelogic.PrintClientHelp()
		case "spam":
			if len(words) < 2 {
				fmt.Printf("int argument missing\n")
				continue
			}

			numS := words[1]
			num, err := strconv.Atoi(numS)
			if err != nil {
				fmt.Printf("argument must be of type int\n")
				continue
			}
			for range num {
				log := gamelogic.GetMaliciousLog()
				err := PublishGameLog(ch, routing.GameLog{
					CurrentTime: time.Now(),
					Message:     log,
					Username:    username,
				})
				if err != nil {
					fmt.Printf("failure publishing spam log: %+v\n", err)
					continue
				}
			}

		case "quit":
			gamelogic.PrintQuit()
			break inf
		default:
			fmt.Printf("cannot parse %s \n", first)
		}
	}

	// sigC := make(chan os.Signal, 1)
	// signal.Notify(sigC, os.Interrupt, syscall.SIGTERM)
	// <-sigC

	fmt.Printf("\nINTERRUPT or KILL signal received, shutting down server\n")
}
