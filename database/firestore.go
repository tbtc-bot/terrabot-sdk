package database

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
	"github.com/tbtc-bot/terrabot"
	"go.uber.org/zap"
	"google.golang.org/api/option"
)

const (
	COLLECTION_STRATEGIES string = "strategies"
	FIELD_BOT_ID          string = "botId"
	FIELD_SYMBOL          string = "symbol"
	FIELD_POSITION_SIDE   string = "positionSide"
	FIELD_STATUS          string = "status"
)

var ctx = context.Background()

type FirestoreDB struct {
	client *firestore.Client
}

func NewFirestoreDB(serviceAccount string) *FirestoreDB {
	sa := option.WithCredentialsFile(serviceAccount)
	app, err := firebase.NewApp(ctx, nil, sa)
	if err != nil {
		fmt.Println(err)
		zap.L().Error("Error initiating firebase",
			zap.String("error", err.Error()),
		)
	}
	client, err := app.Firestore(ctx)
	if err != nil {
		zap.L().Error("Error initiating firestore client",
			zap.String("error", err.Error()),
		)
	}
	return &FirestoreDB{
		client: client,
	}
}

func (fdb *FirestoreDB) UpdateStrategyStatusInFirestore(botID string, symbol string, positionSide terrabot.PositionSideType, strategyStatus terrabot.StrategyStatus) error {
	iter := fdb.client.Collection(COLLECTION_STRATEGIES).
		Where(FIELD_BOT_ID, "==", botID).
		Where(FIELD_SYMBOL, "==", symbol).
		Where(FIELD_POSITION_SIDE, "==", positionSide).
		Documents(ctx)

	doc, err := iter.Next()
	if err != nil {
		return fmt.Errorf("failed to get strategy from firebase %s", err)
	}

	_, err = doc.Ref.Update(ctx, []firestore.Update{{Path: FIELD_STATUS, Value: strategyStatus}})
	if err != nil {
		return fmt.Errorf("failed to update strategy status in firebase %s", err)
	}

	return nil
}
