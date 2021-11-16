package database

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
	"github.com/tbtc-bot/terrabot-sdk"
	"go.uber.org/zap"
	"google.golang.org/api/iterator"
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

func (fdb *FirestoreDB) GetStrategyStatus(botID string, symbol string, positionSide terrabot.PositionSideType) (*terrabot.StrategyStatus, error) {
	iter := fdb.client.Collection(COLLECTION_STRATEGIES).
		Where(FIELD_BOT_ID, "==", botID).
		Where(FIELD_SYMBOL, "==", symbol).
		Where(FIELD_POSITION_SIDE, "==", positionSide).
		Documents(ctx)

	doc, err := iter.Next()
	if err == iterator.Done {
		return nil, fmt.Errorf("strategy not found in firestore %s", err)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get strategy from firestore %s", err)
	}

	data := doc.Data()
	statusString := fmt.Sprintf("%v", data["status"]) // convert interface to string
	status := terrabot.StrategyStatus(statusString)
	return &status, nil
}

func (fdb *FirestoreDB) UpdateStrategyStatus(botID string, symbol string, positionSide terrabot.PositionSideType, strategyStatus terrabot.StrategyStatus) error {
	iter := fdb.client.Collection(COLLECTION_STRATEGIES).
		Where(FIELD_BOT_ID, "==", botID).
		Where(FIELD_SYMBOL, "==", symbol).
		Where(FIELD_POSITION_SIDE, "==", positionSide).
		Documents(ctx)

	doc, err := iter.Next()
	if err == iterator.Done {
		return fmt.Errorf("strategy not found in firestore %s", err)
	}
	if err != nil {
		return fmt.Errorf("failed to get strategy from firestore %s", err)
	}

	_, err = doc.Ref.Update(ctx, []firestore.Update{{Path: FIELD_STATUS, Value: strategyStatus}})
	if err != nil {
		return fmt.Errorf("failed to update strategy status in firestore %s", err)
	}

	return nil
}
