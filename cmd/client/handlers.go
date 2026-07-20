package main

import (
	"fmt"
	"time"

	"github.com/mattnickolaus/learn-pub-sub-starter/internal/gamelogic"
	"github.com/mattnickolaus/learn-pub-sub-starter/internal/pubsub"
	"github.com/mattnickolaus/learn-pub-sub-starter/internal/routing"

	amqp "github.com/rabbitmq/amqp091-go"
)

func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) pubsub.AckType {
	return func(ps routing.PlayingState) pubsub.AckType {
		defer fmt.Print("> ")
		gs.HandlePause(ps)
		return pubsub.Ack
	}
}

func handlerArmyMoves(gs *gamelogic.GameState, ch *amqp.Channel) func(move gamelogic.ArmyMove) pubsub.AckType {
	return func(move gamelogic.ArmyMove) pubsub.AckType {
		defer fmt.Print("> ")
		outcome := gs.HandleMove(move)

		switch outcome {
		case gamelogic.MoveOutComeSafe:
			return pubsub.Ack
		case gamelogic.MoveOutcomeMakeWar:
			err := pubsub.PublishJSON(
				ch,
				routing.ExchangePerilTopic,
				routing.WarRecognitionsPrefix+"."+gs.Player.Username,
				gamelogic.RecognitionOfWar{
					Attacker: move.Player,
					Defender: gs.GetPlayerSnap(),
				},
			)
			if err != nil {
				fmt.Printf("Error publishing war recognition msg: %v\n", err)
				return pubsub.NackRequeue
			}
			return pubsub.Ack
		case gamelogic.MoveOutcomeSamePlayer:
			return pubsub.NackDiscard
		default:
			return pubsub.NackDiscard
		}
	}
}

func handlerConsumeWar(gs *gamelogic.GameState, ch *amqp.Channel) func(warRec gamelogic.RecognitionOfWar) pubsub.AckType {
	return func(warRec gamelogic.RecognitionOfWar) pubsub.AckType {
		defer fmt.Print("> ")

		gl := routing.GameLog{
			CurrentTime: time.Now(),
			Username:    gs.Player.Username,
		}

		outcome, winner, loser := gs.HandleWar(warRec)
		switch outcome {
		case gamelogic.WarOutcomeNotInvolved:
			return pubsub.NackRequeue
		case gamelogic.WarOutcomeNoUnits:
			return pubsub.NackDiscard
		case gamelogic.WarOutcomeOpponentWon:
			gl.Message = fmt.Sprintf("%s won against %s", winner, loser)
			break
		case gamelogic.WarOutcomeYouWon:
			gl.Message = fmt.Sprintf("%s won against %s", winner, loser)
			break
		case gamelogic.WarOutcomeDraw:
			gl.Message = fmt.Sprintf("%s and %s resulted in a draw", winner, loser)
			break
		default:
			fmt.Printf("Error no valid war outcome\n")
			return pubsub.NackDiscard
		}

		return publishGameLog(gl, ch)
	}
}

func publishGameLog(gl routing.GameLog, ch *amqp.Channel) pubsub.AckType {
	err := pubsub.PublishGob(
		ch,
		routing.ExchangePerilTopic,
		routing.GameLogSlug+"."+gl.Username,
		gl,
	)
	if err != nil {
		return pubsub.NackRequeue
	}

	return pubsub.Ack
}
