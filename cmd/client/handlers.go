package main

import (
	"fmt"
	"time"

	"github.com/rabbitmq/amqp091-go"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
)

func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) pubsub.AckType {
	return func(st routing.PlayingState) pubsub.AckType {
		defer fmt.Printf("> ")
		gs.HandlePause(st)
		return pubsub.Ack
	}
}

func handlerArmyMove(gs *gamelogic.GameState, ch *amqp091.Channel) func(gamelogic.ArmyMove) pubsub.AckType {
	return func(mv gamelogic.ArmyMove) pubsub.AckType {
		defer fmt.Printf("> ")
		outcome := gs.HandleMove(mv)
		switch outcome {
		case gamelogic.MoveOutComeSafe:
			return pubsub.Ack
		case gamelogic.MoveOutcomeMakeWar:
			err := pubsub.PublishJSON(ch, routing.ExchangePerilTopic, fmt.Sprintf("%s.%s", routing.WarRecognitionsPrefix, mv.Player.Username), gamelogic.RecognitionOfWar{
				Attacker: mv.Player,
				Defender: gs.Player,
			})
			if err != nil {
				fmt.Printf("%s\n", err.Error())
				return pubsub.NackReque
			}
			return pubsub.Ack
		default:
			return pubsub.NackDiscard
		}
	}
}

func handlerWar(gs *gamelogic.GameState, ch *amqp091.Channel) func(gamelogic.RecognitionOfWar) pubsub.AckType {
	return func(r gamelogic.RecognitionOfWar) pubsub.AckType {
		defer fmt.Printf("> ")
		outcome, winner, loser := gs.HandleWar(r)

		acky := pubsub.Ack
		YouWon := false
		YouLost := false
		Draw := false

		switch outcome {
		case gamelogic.WarOutcomeNotInvolved:
			acky = pubsub.NackReque
		case gamelogic.WarOutcomeNoUnits:
			acky = pubsub.NackDiscard
		case gamelogic.WarOutcomeOpponentWon:
			YouLost = true
			acky = pubsub.Ack
		case gamelogic.WarOutcomeYouWon:
			YouWon = true
			acky = pubsub.Ack
		case gamelogic.WarOutcomeDraw:
			Draw = true
			acky = pubsub.Ack
		default:
			fmt.Printf("could not parse war-outcome\n")
			return pubsub.NackDiscard
		}

		log := ""
		if YouWon || YouLost {
			log = fmt.Sprintf("%s won a war against %s", winner, loser)
		}
		if Draw {
			log = fmt.Sprintf("A war between %s and %s resulted in a draw", winner, loser)
		}

		if log != "" {
			err := PublishGameLog(ch, routing.GameLog{
				CurrentTime: time.Now(),
				Message:     log,
				Username:    gs.Player.Username,
			})
			if err != nil {
				acky = pubsub.NackReque
			}
		}
		return acky
	}
}

func PublishGameLog(ch *amqp091.Channel, log routing.GameLog) error {
	// TODO: should not be hardcoded!
	exchange := routing.ExchangePerilTopic
	key := fmt.Sprintf("%s.%s", routing.GameLogSlug, log.Username)
	err := pubsub.PublishGob(ch, exchange, key, log)
	// err := pubsub.PublishJSON(ch, exchange, key, log)
	return err
}
