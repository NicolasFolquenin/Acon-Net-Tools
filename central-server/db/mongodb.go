package db

import (
	"context"
	"fmt"
	"log"
	"time"

	"central-server/config"
	"central-server/models"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// MongoDBClient encapsula a conexão com o MongoDB e a coleção Time Series
type MongoDBClient struct {
	Client     *mongo.Client
	Database   *mongo.Database
	Collection *mongo.Collection
	cfg        *config.AppConfig
}

// InitMongoDB conecta ao cluster MongoDB e assegura a coleção Time Series
func InitMongoDB(ctx context.Context, cfg *config.AppConfig) (*MongoDBClient, error) {
	clientOptions := options.Client().ApplyURI(cfg.MongoURI)

	log.Printf("[MONGODB] Conectando ao MongoDB em: %s...", cfg.MongoURI)
	client, err := mongo.Connect(clientOptions)
	if err != nil {
		return nil, fmt.Errorf("erro ao conectar no MongoDB: %w", err)
	}

	// Ping de verificação
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx, nil); err != nil {
		log.Printf("[MONGODB][AVISO] Não foi possível verificar o ping inicial: %v (Tentará novamente durante as operações)", err)
	} else {
		log.Println("[MONGODB] Conexão com MongoDB validada com sucesso via Ping.")
	}

	database := client.Database(cfg.DatabaseName)
	collection := database.Collection(cfg.CollectionName)

	m := &MongoDBClient{
		Client:     client,
		Database:   database,
		Collection: collection,
		cfg:        cfg,
	}

	// Assegura criação da coleção Time Series
	if err := m.ensureTimeSeriesCollection(ctx); err != nil {
		log.Printf("[MONGODB][AVISO] Inicialização da Time Series collection: %v", err)
	}

	return m, nil
}

// ensureTimeSeriesCollection cria a coleção Time Series caso ainda não exista:
// timeField: "timestamp", metaField: "metadata", granularity: "hours"
func (m *MongoDBClient) ensureTimeSeriesCollection(ctx context.Context) error {
	// Verifica se a coleção já existe
	names, err := m.Database.ListCollectionNames(ctx, bson.M{"name": m.cfg.CollectionName})
	if err == nil && len(names) > 0 {
		log.Printf("[MONGODB] Coleção Time Series '%s' já existe.", m.cfg.CollectionName)
		return nil
	}

	log.Printf("[MONGODB] Criando coleção Time Series '%s.%s' (timeField: timestamp, metaField: metadata, granularity: hours)...",
		m.cfg.DatabaseName, m.cfg.CollectionName)

	// Criação via comando MongoDB padrão (compatível com todas as versões 5.0+)
	createCmd := bson.D{
		{Key: "create", Value: m.cfg.CollectionName},
		{Key: "timeseries", Value: bson.D{
			{Key: "timeField", Value: "timestamp"},
			{Key: "metaField", Value: "metadata"},
			{Key: "granularity", Value: "hours"},
		}},
	}

	var result bson.M
	if err := m.Database.RunCommand(ctx, createCmd).Decode(&result); err != nil {
		return fmt.Errorf("falha ao criar Time Series collection: %w", err)
	}

	log.Println("[MONGODB] Coleção Time Series criada com sucesso!")

	// Cria índices secundários para pesquisas ultra rápidas
	indexModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "metadata.node_id", Value: 1},
			{Key: "timestamp", Value: -1},
		},
	}
	_, _ = m.Collection.Indexes().CreateOne(ctx, indexModel)

	return nil
}

// InsertMetric insere uma nova medição no MongoDB Time Series
func (m *MongoDBClient) InsertMetric(ctx context.Context, payload *models.MetricPayloadInput) error {
	doc := models.TimeSeriesMetricDocument{
		Timestamp: payload.Timestamp.UTC(),
		Metadata: models.NodeMetadata{
			NodeID:   payload.NodeID,
			Location: payload.Location,
		},
		DownloadMbps: payload.DownloadMbps,
		UploadMbps:   payload.UploadMbps,
		PingGateway:  payload.PingGateway,
	}

	_, err := m.Collection.InsertOne(ctx, doc)
	if err != nil {
		return fmt.Errorf("erro ao inserir documento no MongoDB: %w", err)
	}

	return nil
}

// GetMetricsLast24Hours busca os dados das últimas 24 horas ordenados por timestamp
func (m *MongoDBClient) GetMetricsLast24Hours(ctx context.Context, nodeIDFilter string, hoursLimit int) ([]models.FlattenedMetricItem, error) {
	if hoursLimit <= 0 {
		hoursLimit = 24
	}
	since := time.Now().UTC().Add(-time.Duration(hoursLimit) * time.Hour)

	filter := bson.M{
		"timestamp": bson.M{"$gte": since},
	}

	if nodeIDFilter != "" && nodeIDFilter != "all" {
		filter["metadata.node_id"] = nodeIDFilter
	}

	findOpts := options.Find().SetSort(bson.D{{Key: "timestamp", Value: 1}}) // Ordenação cronológica

	cursor, err := m.Collection.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, fmt.Errorf("falha na query MongoDB: %w", err)
	}
	defer cursor.Close(ctx)

	var results []models.FlattenedMetricItem
	for cursor.Next(ctx) {
		var doc models.TimeSeriesMetricDocument
		if err := cursor.Decode(&doc); err != nil {
			log.Printf("[MONGODB] Erro ao decodificar documento: %v", err)
			continue
		}

		results = append(results, models.FlattenedMetricItem{
			NodeID:       doc.Metadata.NodeID,
			Location:     doc.Metadata.Location,
			Timestamp:    doc.Timestamp,
			DownloadMbps: doc.DownloadMbps,
			UploadMbps:   doc.UploadMbps,
			PingGateway:  doc.PingGateway,
		})
	}

	return results, nil
}

// GetDistinctNodes retorna a lista de todos os nós registrados
func (m *MongoDBClient) GetDistinctNodes(ctx context.Context) ([]string, error) {
	results, err := m.Collection.Distinct(ctx, "metadata.node_id", bson.M{})
	if err != nil {
		return nil, err
	}

	var nodes []string
	for _, val := range results {
		if s, ok := val.(string); ok && s != "" {
			nodes = append(nodes, s)
		}
	}
	return nodes, nil
}

// Disconnect encerra a conexão com o MongoDB
func (m *MongoDBClient) Disconnect(ctx context.Context) error {
	if m.Client != nil {
		return m.Client.Disconnect(ctx)
	}
	return nil
}
