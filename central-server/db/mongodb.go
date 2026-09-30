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
	Client          *mongo.Client
	Database        *mongo.Database
	Collection         *mongo.Collection
	UsersCollection    *mongo.Collection
	GroupsCollection   *mongo.Collection
	MachinesCollection *mongo.Collection
	cfg                *config.AppConfig
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
	usersCollection := database.Collection("users")
	groupsCollection := database.Collection("machine_groups")
	machinesCollection := database.Collection("machines")

	m := &MongoDBClient{
		Client:             client,
		Database:           database,
		Collection:         collection,
		UsersCollection:    usersCollection,
		GroupsCollection:   groupsCollection,
		MachinesCollection: machinesCollection,
		cfg:                cfg,
	}

	// Assegura criação da coleção Time Series
	if err := m.ensureTimeSeriesCollection(ctx); err != nil {
		log.Printf("[MONGODB][AVISO] Inicialização da Time Series collection: %v", err)
	}

	// Assegura índice único no campo username para a coleção users
	_, _ = usersCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "username", Value: 1}},
		Options: options.Index().SetUnique(true),
	})

	// Assegura grupos padrão no sistema
	if err := m.SeedInitialGroups(ctx); err != nil {
		log.Printf("[MONGODB][AVISO] Falha ao verificar/semear grupos padrão: %v", err)
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

	// Criação via comando MongoDB padrão
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
		log.Printf("[MONGODB][AVISO] Engine Time Series não disponível (%v). Criando coleção padrão com índices...", err)
		if errCreate := m.Database.CreateCollection(ctx, m.cfg.CollectionName); errCreate != nil {
			log.Printf("[MONGODB] Coleção padrão já pronta ou criada: %v", errCreate)
		}
	} else {
		log.Println("[MONGODB] Coleção Time Series criada com sucesso!")
	}

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
		DownloadMbps:   payload.DownloadMbps,
		UploadMbps:     payload.UploadMbps,
		PingGateway:    payload.PingGateway,
		ProfileVersion: payload.ProfileVersion,
		Metrics:        payload.Metrics,
		SystemInfo:     payload.SystemInfo,
		SwitchInfo:     payload.SwitchInfo,
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
			NodeID:         doc.Metadata.NodeID,
			Location:       doc.Metadata.Location,
			Timestamp:      doc.Timestamp,
			DownloadMbps:   doc.DownloadMbps,
			UploadMbps:     doc.UploadMbps,
			PingGateway:    doc.PingGateway,
			ProfileVersion: doc.ProfileVersion,
			Metrics:        doc.Metrics,
			SystemInfo:     doc.SystemInfo,
			SwitchInfo:     doc.SwitchInfo,
		})
	}

	return results, nil
}

// GetDistinctNodes retorna a lista de todos os nós registrados
func (m *MongoDBClient) GetDistinctNodes(ctx context.Context) ([]string, error) {
	res := m.Collection.Distinct(ctx, "metadata.node_id", bson.M{})
	if err := res.Err(); err != nil {
		return nil, err
	}

	var nodes []string
	if err := res.Decode(&nodes); err != nil {
		return nil, err
	}

	return nodes, nil
}

// DeleteNode remove todos os registros e métricas pertencentes a um determinado nó
func (m *MongoDBClient) DeleteNode(ctx context.Context, nodeID string) (int64, error) {
	if m.Collection == nil {
		return 0, fmt.Errorf("colecao mongodb nao inicializada")
	}

	filter := bson.M{
		"metadata.node_id": nodeID,
	}

	res, err := m.Collection.DeleteMany(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("erro ao excluir no '%s': %w", nodeID, err)
	}

	log.Printf("[MONGODB] No '%s' excluido do sistema. Total de registros removidos: %d", nodeID, res.DeletedCount)
	return res.DeletedCount, nil
}

// GetUserByUsername busca um usuário pelo nome de usuário
func (m *MongoDBClient) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	if m.UsersCollection == nil {
		return nil, fmt.Errorf("colecao de usuarios nao inicializada")
	}

	var user models.User
	err := m.UsersCollection.FindOne(ctx, bson.M{"username": username}).Decode(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// CreateUser insere um novo usuário na coleção
func (m *MongoDBClient) CreateUser(ctx context.Context, user *models.User) error {
	if m.UsersCollection == nil {
		return fmt.Errorf("colecao de usuarios nao inicializada")
	}

	_, err := m.UsersCollection.InsertOne(ctx, user)
	if err != nil {
		return fmt.Errorf("erro ao inserir usuario no MongoDB: %w", err)
	}

	log.Printf("[MONGODB] Novo usuario cadastrado: %s (perfil: %s)", user.Username, user.Role)
	return nil
}

// ListUsers retorna todos os usuários cadastrados no banco
func (m *MongoDBClient) ListUsers(ctx context.Context) ([]models.User, error) {
	if m.UsersCollection == nil {
		return nil, fmt.Errorf("colecao de usuarios nao inicializada")
	}

	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := m.UsersCollection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, fmt.Errorf("falha ao listar usuarios: %w", err)
	}
	defer cursor.Close(ctx)

	var users []models.User
	for cursor.Next(ctx) {
		var u models.User
		if err := cursor.Decode(&u); err != nil {
			continue
		}
		users = append(users, u)
	}

	return users, nil
}

// DeleteUser remove um usuário pelo nome de usuário
func (m *MongoDBClient) DeleteUser(ctx context.Context, username string) error {
	if m.UsersCollection == nil {
		return fmt.Errorf("colecao de usuarios nao inicializada")
	}

	res, err := m.UsersCollection.DeleteOne(ctx, bson.M{"username": username})
	if err != nil {
		return fmt.Errorf("erro ao deletar usuario '%s': %w", username, err)
	}
	if res.DeletedCount == 0 {
		return fmt.Errorf("usuario '%s' nao encontrado", username)
	}

	log.Printf("[MONGODB] Usuario '%s' removido com sucesso", username)
	return nil
}

// UpdateUserRole altera a permissão/papel de um usuário no sistema (admin ou operator)
func (m *MongoDBClient) UpdateUserRole(ctx context.Context, username string, newRole string) error {
	if m.UsersCollection == nil {
		return fmt.Errorf("colecao de usuarios nao inicializada")
	}

	filter := bson.M{"username": username}
	update := bson.M{"$set": bson.M{"role": newRole}}

	res, err := m.UsersCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("erro ao atualizar perfil do usuario '%s': %w", username, err)
	}
	if res.MatchedCount == 0 {
		return fmt.Errorf("usuario '%s' nao encontrado", username)
	}

	log.Printf("[MONGODB] Perfil do usuario '%s' atualizado para '%s'", username, newRole)
	return nil
}

// UpsertKeycloakUser sincroniza um usuário que autenticou via Keycloak/OIDC:
// Se já existe, atualiza LastLogin e preserva a Role já definida pelo Admin;
// Se não existe, insere o novo usuário com a Role padrão fornecida.
func (m *MongoDBClient) UpsertKeycloakUser(ctx context.Context, user *models.User) (*models.User, error) {
	if m.UsersCollection == nil {
		return nil, fmt.Errorf("colecao de usuarios nao inicializada")
	}

	existing, err := m.GetUserByUsername(ctx, user.Username)
	if err == nil && existing != nil {
		// Atualiza dados de login e preserva Role configurada no MongoDB!
		updateSet := bson.M{
			"last_login": user.LastLogin,
			"auth_type":  "keycloak",
		}
		if user.Email != "" {
			updateSet["email"] = user.Email
		}
		if user.FullName != "" {
			updateSet["full_name"] = user.FullName
		}

		_, _ = m.UsersCollection.UpdateOne(ctx, bson.M{"username": user.Username}, bson.M{"$set": updateSet})
		existing.LastLogin = user.LastLogin
		return existing, nil
	}

	// Não existe, cria novo usuário Keycloak
	user.AuthType = "keycloak"
	user.CreatedAt = time.Now().UTC()
	_, err = m.UsersCollection.InsertOne(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("erro ao registrar usuario keycloak: %w", err)
	}

	log.Printf("[MONGODB] Novo usuario Keycloak registrado: %s (perfil: %s)", user.Username, user.Role)
	return user, nil
}

// Disconnect encerra a conexão com o MongoDB
func (m *MongoDBClient) Disconnect(ctx context.Context) error {
	if m.Client != nil {
		return m.Client.Disconnect(ctx)
	}
	return nil
}

// SeedInitialGroups assegura que os grupos padrão (default, reitoria, acon) existam no MongoDB
func (m *MongoDBClient) SeedInitialGroups(ctx context.Context) error {
	if m.GroupsCollection == nil {
		return nil
	}

	initialGroups := models.GetInitialGroups()
	for _, grp := range initialGroups {
		count, err := m.GroupsCollection.CountDocuments(ctx, bson.M{"_id": grp.ID})
		if err != nil {
			continue
		}
		if count == 0 {
			_, err = m.GroupsCollection.InsertOne(ctx, grp)
			if err != nil {
				log.Printf("[MONGODB][AVISO] Falha ao semear grupo '%s': %v", grp.ID, err)
			} else {
				log.Printf("[MONGODB] Grupo inicial '%s' (%s) semeado com sucesso.", grp.ID, grp.Name)
			}
		}
	}
	return nil
}

// GetGroup busca um grupo de máquinas pelo ID
func (m *MongoDBClient) GetGroup(ctx context.Context, id string) (*models.MachineGroup, error) {
	if m.GroupsCollection == nil {
		return nil, fmt.Errorf("colecao de grupos nao inicializada")
	}

	var grp models.MachineGroup
	err := m.GroupsCollection.FindOne(ctx, bson.M{"_id": id}).Decode(&grp)
	if err != nil {
		return nil, err
	}
	return &grp, nil
}

// ListGroups retorna todos os grupos de máquinas cadastrados
func (m *MongoDBClient) ListGroups(ctx context.Context) ([]models.MachineGroup, error) {
	if m.GroupsCollection == nil {
		return nil, fmt.Errorf("colecao de grupos nao inicializada")
	}

	cursor, err := m.GroupsCollection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var groups []models.MachineGroup
	if err := cursor.All(ctx, &groups); err != nil {
		return nil, err
	}
	return groups, nil
}

// SaveGroup cria ou atualiza um grupo de máquinas
func (m *MongoDBClient) SaveGroup(ctx context.Context, grp *models.MachineGroup) error {
	if m.GroupsCollection == nil {
		return fmt.Errorf("colecao de grupos nao inicializada")
	}

	grp.UpdatedAt = time.Now().UTC()
	opts := options.Update().SetUpsert(true)
	_, err := m.GroupsCollection.UpdateOne(ctx, bson.M{"_id": grp.ID}, bson.M{"$set": grp}, opts)
	return err
}

// GetMachine busca o registro de uma máquina pelo hostname
func (m *MongoDBClient) GetMachine(ctx context.Context, hostname string) (*models.MachineRegistration, error) {
	if m.MachinesCollection == nil {
		return nil, fmt.Errorf("colecao de maquinas nao inicializada")
	}

	var mach models.MachineRegistration
	err := m.MachinesCollection.FindOne(ctx, bson.M{"_id": hostname}).Decode(&mach)
	if err != nil {
		return nil, err
	}
	return &mach, nil
}

// ListMachines retorna todas as máquinas registradas
func (m *MongoDBClient) ListMachines(ctx context.Context) ([]models.MachineRegistration, error) {
	if m.MachinesCollection == nil {
		return nil, fmt.Errorf("colecao de maquinas nao inicializada")
	}

	cursor, err := m.MachinesCollection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var machines []models.MachineRegistration
	if err := cursor.All(ctx, &machines); err != nil {
		return nil, err
	}
	return machines, nil
}

// SaveMachine cria ou atualiza o registro de uma máquina
func (m *MongoDBClient) SaveMachine(ctx context.Context, mach *models.MachineRegistration) error {
	if m.MachinesCollection == nil {
		return fmt.Errorf("colecao de maquinas nao inicializada")
	}

	mach.UpdatedAt = time.Now().UTC()
	mach.LastSeen = time.Now().UTC()
	if mach.CreatedAt.IsZero() {
		mach.CreatedAt = mach.UpdatedAt
	}

	opts := options.Update().SetUpsert(true)
	_, err := m.MachinesCollection.UpdateOne(ctx, bson.M{"_id": mach.Hostname}, bson.M{"$set": mach}, opts)
	return err
}

// UpdateMachine atualiza o grupo, testes customizados ou status de uma máquina
func (m *MongoDBClient) UpdateMachine(ctx context.Context, hostname string, req *models.UpdateMachineRequest) error {
	if m.MachinesCollection == nil {
		return fmt.Errorf("colecao de maquinas nao inicializada")
	}

	update := bson.M{
		"updated_at": time.Now().UTC(),
	}
	if req.GroupID != "" {
		update["group_id"] = req.GroupID
	}
	update["is_custom"] = req.IsCustom
	if req.IsCustom {
		update["custom_tests"] = req.CustomTests
	} else {
		update["custom_tests"] = nil
	}
	if req.Status != "" {
		update["status"] = req.Status
	}

	res, err := m.MachinesCollection.UpdateOne(ctx, bson.M{"_id": hostname}, bson.M{"$set": update})
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return fmt.Errorf("maquina '%s' nao encontrada", hostname)
	}
	return nil
}


