package auction_test

import (
	"context"
	"fullcycle-auction_go/internal/entity/auction_entity"
	"fullcycle-auction_go/internal/infra/database/auction"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func getTestDB(t *testing.T) *mongo.Database {

	// Cria o URI de conexão com o MongoDB no Docker
	uri := "mongodb://admin:admin@mongodb:27017/auctions?authSource=admin"

	// Conecta ao MongoDB
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(uri))
	if err != nil {
		t.Fatalf("Erro ao conectar ao MongoDB: %v", err)
	}

	return client.Database("auctions")
}

func TestCreateAuction_ShouldCreateAndAutoClose(t *testing.T) {
	// Obtém o banco de dados de teste
	db := getTestDB(t)
	repo := auction.NewAuctionRepository(db)

	// Limpa a coleção de leilões antes de rodar o teste
	_ = repo.Collection.Drop(context.Background())

	// Define o tempo de expiração para 1 minuto (variável já configurada no Docker Compose)
	os.Setenv("AUCTION_DURATION", "1")

	// Cria o leilão
	startTime := time.Now()
	auctionEntity := &auction_entity.Auction{
		Id:          "auto-expire-auction-001",
		ProductName: "Expiring Test Product",
		Category:    "Gadgets",
		Description: "Temporary test item",
		Condition:   auction_entity.New,
		Status:      auction_entity.Active,
		Timestamp:   startTime,
	}

	// Chama a função CreateAuction que irá disparar expirationCheck em uma goroutine
	err := repo.CreateAuction(context.Background(), auctionEntity)
	assert.Nil(t, err)

	// Verifica se o leilão foi inserido corretamente no banco
	var created auction.AuctionEntityMongo
	errFind := repo.Collection.FindOne(context.Background(), map[string]interface{}{"_id": auctionEntity.Id}).Decode(&created)
	assert.Nil(t, errFind)
	assert.Equal(t, auction_entity.Active, created.Status)

	// Espera um pouco para a goroutine de expiração rodar
	time.Sleep(2 * time.Second)

	// Verifica se o leilão foi automaticamente fechado após o tempo de expiração
	var updated auction.AuctionEntityMongo
	errFind = repo.Collection.FindOne(context.Background(), map[string]interface{}{"_id": auctionEntity.Id}).Decode(&updated)
	assert.Nil(t, errFind)
	assert.Equal(t, auction_entity.Completed, updated.Status)
}
