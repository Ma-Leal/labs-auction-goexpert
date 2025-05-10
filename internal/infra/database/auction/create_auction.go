package auction

import (
	"context"
	"fullcycle-auction_go/configuration/logger"
	"fullcycle-auction_go/internal/entity/auction_entity"
	"fullcycle-auction_go/internal/internal_error"
	"os"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type AuctionEntityMongo struct {
	Id          string                          `bson:"_id"`
	ProductName string                          `bson:"product_name"`
	Category    string                          `bson:"category"`
	Description string                          `bson:"description"`
	Condition   auction_entity.ProductCondition `bson:"condition"`
	Status      auction_entity.AuctionStatus    `bson:"status"`
	Timestamp   int64                           `bson:"timestamp"`
}
type AuctionRepository struct {
	Collection *mongo.Collection
}

func NewAuctionRepository(database *mongo.Database) *AuctionRepository {
	return &AuctionRepository{
		Collection: database.Collection("auctions"),
	}
}

func (ar *AuctionRepository) CreateAuction(
	ctx context.Context,
	auctionEntity *auction_entity.Auction) *internal_error.InternalError {
	auctionEntityMongo := &AuctionEntityMongo{
		Id:          auctionEntity.Id,
		ProductName: auctionEntity.ProductName,
		Category:    auctionEntity.Category,
		Description: auctionEntity.Description,
		Condition:   auctionEntity.Condition,
		Status:      auctionEntity.Status,
		Timestamp:   auctionEntity.Timestamp.Unix(),
	}
	_, err := ar.Collection.InsertOne(ctx, auctionEntityMongo)
	if err != nil {
		logger.Error("Error trying to insert auction", err)
		return internal_error.NewInternalServerError("Error trying to insert auction")
	}

	go ar.expirationCheck(auctionEntity.Id, auctionEntity.Timestamp)

	return nil
}

func (ar *AuctionRepository) expirationCheck(auctionId string, startTime time.Time) {
	durationEnv := os.Getenv("AUCTION_DURATION")

	durationMinutes, err := strconv.Atoi(durationEnv)
	if err != nil {
		logger.Error("Failed to parse AUCTION_DURATION", err)
		return
	}

	expirationTime := startTime.Add(time.Duration(durationMinutes) * time.Minute)
	waitDuration := time.Until(expirationTime)

	if waitDuration > 0 {
		time.Sleep(waitDuration)
	}

	if err := ar.auctionClose(context.Background(), auctionId); err != nil {
		logger.Error("Failed to close expired auction", err)
	}
}

func (ar *AuctionRepository) auctionClose(ctx context.Context, auctionId string) error {
	filter := bson.M{
		"_id":    auctionId,
		"status": auction_entity.Active,
	}
	update := bson.M{
		"$set": bson.M{"status": auction_entity.Completed},
	}
	_, err := ar.Collection.UpdateOne(ctx, filter, update)
	return err
}
