package main

import (
	"fmt"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
)

func handlerLogs() func(routing.GameLog) pubsub.AckType {
	return func(log routing.GameLog) pubsub.AckType {
		defer fmt.Printf("> ")
		err := gamelogic.WriteLog(log)
		if err != nil {
			fmt.Printf("Failed writing log: %+v\n", err)
		}
		return pubsub.Ack
	}
}
