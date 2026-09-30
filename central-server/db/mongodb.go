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
	Client           *mongo.Client
	Database         *mongo.Database
	Collection       *mongo.Collection
	UsersCollection  *mongo.Collection
	GroupsCollection *mongo.Collection
	cfg              *config.AppConfig
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
	groupsCollection := database.Collection("test_groups")

	m := &MongoDBClient{
		Client:           client,
		Database:         database,
		Collection:       collection,
		UsersCollection:  usersCollection,
		GroupsCollection: groupsCollection,
		cfg:              cfg,
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

	// Assegura grupo padrão para testes e sondas
	if err := m.ensureDefaultGroup(ctx); err != nil {
		log.Printf("[MONGODB][AVISO] Falha ao assegurar grupo padrão: %v", err)
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
			NodeID:    payload.NodeID,
			Location:  payload.Location,
			GroupName: payload.GroupName,
			VLAN:      payload.VLAN,
		},
		DownloadMbps:   payload.DownloadMbps,
		UploadMbps:     payload.UploadMbps,
		PingGateway:    payload.PingGateway,
		SystemInfo:     payload.SystemInfo,
		SwitchInfo:     payload.SwitchInfo,
		ModularResults: payload.ModularResults,
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
			GroupName:      doc.Metadata.GroupName,
			VLAN:           doc.Metadata.VLAN,
			Timestamp:      doc.Timestamp,
			DownloadMbps:   doc.DownloadMbps,
			UploadMbps:     doc.UploadMbps,
			PingGateway:    doc.PingGateway,
			SystemInfo:     doc.SystemInfo,
			SwitchInfo:     doc.SwitchInfo,
			ModularResults: doc.ModularResults,
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

type rawTestGroup struct {
	ID              interface{}              `bson:"_id,omitempty"`
	Name            string                   `bson:"name"`
	Description     string                   `bson:"description"`
	VLANs           []int                    `bson:"vlans"`
	IsDefault       bool                     `bson:"is_default"`
	IntervalMinutes int                      `bson:"interval_minutes"`
	Modules         models.TestModulesConfig `bson:"modules"`
	CreatedAt       time.Time                `bson:"created_at"`
	UpdatedAt       time.Time                `bson:"updated_at"`
}

func rawToTestGroup(r *rawTestGroup) models.TestGroup {
	g := models.TestGroup{
		Name:            r.Name,
		Description:     r.Description,
		VLANs:           r.VLANs,
		IsDefault:       r.IsDefault,
		IntervalMinutes: r.IntervalMinutes,
		Modules:         r.Modules,
		CreatedAt:       r.CreatedAt,
		UpdatedAt:       r.UpdatedAt,
	}
	if g.VLANs == nil {
		g.VLANs = []int{}
	}
	if g.IntervalMinutes <= 0 {
		g.IntervalMinutes = 60
	}
	switch v := r.ID.(type) {
	case string:
		g.ID = v
	case bson.ObjectID:
		g.ID = v.Hex()
	default:
		if v != nil {
			g.ID = fmt.Sprintf("%v", v)
		}
	}
	return g
}

func groupIDFilter(idStr string) bson.M {
	if oid, err := bson.ObjectIDFromHex(idStr); err == nil {
		return bson.M{"$or": []bson.M{
			{"_id": idStr},
			{"_id": oid},
		}}
	}
	return bson.M{"_id": idStr}
}

// ensureDefaultGroup cria um grupo padrão corporativo caso a coleção esteja vazia
func (m *MongoDBClient) ensureDefaultGroup(ctx context.Context) error {
	if m.GroupsCollection == nil {
		return nil
	}
	count, err := m.GroupsCollection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	defaultGrp := models.TestGroup{
		Name:            "Grupo Padrão Corporativo",
		Description:     "Grupo padrão aplicado automaticamente para VLANs não cadastradas",
		VLANs:           []int{},
		IsDefault:       true,
		IntervalMinutes: 60,
		Modules:         models.DefaultModulesConfig(),
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}

	newID := fmt.Sprintf("grp-%d", time.Now().UnixNano())
	doc := bson.M{
		"_id":              newID,
		"name":             defaultGrp.Name,
		"description":      defaultGrp.Description,
		"vlans":            defaultGrp.VLANs,
		"is_default":       defaultGrp.IsDefault,
		"interval_minutes": defaultGrp.IntervalMinutes,
		"modules":          defaultGrp.Modules,
		"created_at":       defaultGrp.CreatedAt,
		"updated_at":       defaultGrp.UpdatedAt,
	}

	_, err = m.GroupsCollection.InsertOne(ctx, doc)
	if err != nil {
		return fmt.Errorf("erro ao inserir grupo padrão: %w", err)
	}

	log.Printf("[MONGODB] Grupo padrão de testes inicializado com sucesso (ID: %s)", newID)
	return nil
}

// GetGroupByVLAN busca o grupo que atende à VLAN especificada
func (m *MongoDBClient) GetGroupByVLAN(ctx context.Context, vlan int) (*models.TestGroup, error) {
	if m.GroupsCollection == nil {
		return nil, fmt.Errorf("colecao de grupos nao inicializada")
	}

	if vlan > 0 {
		filter := bson.M{"vlans": vlan}
		var raw rawTestGroup
		err := m.GroupsCollection.FindOne(ctx, filter).Decode(&raw)
		if err == nil {
			grp := rawToTestGroup(&raw)
			return &grp, nil
		}
	}

	// Se não encontrou pela VLAN, busca o grupo padrão
	return m.GetDefaultGroup(ctx)
}

// GetDefaultGroup busca o grupo marcado com is_default: true
func (m *MongoDBClient) GetDefaultGroup(ctx context.Context) (*models.TestGroup, error) {
	if m.GroupsCollection == nil {
		return nil, fmt.Errorf("colecao de grupos nao inicializada")
	}

	var raw rawTestGroup
	err := m.GroupsCollection.FindOne(ctx, bson.M{"is_default": true}).Decode(&raw)
	if err == nil {
		grp := rawToTestGroup(&raw)
		return &grp, nil
	}

	// Fallback se não houver grupo marcado como is_default
	var firstRaw rawTestGroup
	errFirst := m.GroupsCollection.FindOne(ctx, bson.M{}).Decode(&firstRaw)
	if errFirst == nil {
		grp := rawToTestGroup(&firstRaw)
		return &grp, nil
	}

	// Fallback em memória se a coleção estiver vazia
	def := models.TestGroup{
		ID:              "grp-default-fallback",
		Name:            "Grupo Padrão Corporativo",
		Description:     "Grupo padrão fallback",
		VLANs:           []int{},
		IsDefault:       true,
		IntervalMinutes: 60,
		Modules:         models.DefaultModulesConfig(),
	}
	return &def, nil
}

// ListGroups retorna todos os grupos ordenados por nome
func (m *MongoDBClient) ListGroups(ctx context.Context) ([]models.TestGroup, error) {
	if m.GroupsCollection == nil {
		return nil, fmt.Errorf("colecao de grupos nao inicializada")
	}

	findOpts := options.Find().SetSort(bson.D{{Key: "is_default", Value: -1}, {Key: "name", Value: 1}})
	cursor, err := m.GroupsCollection.Find(ctx, bson.M{}, findOpts)
	if err != nil {
		return nil, fmt.Errorf("falha ao listar grupos: %w", err)
	}
	defer cursor.Close(ctx)

	var groups []models.TestGroup
	for cursor.Next(ctx) {
		var raw rawTestGroup
		if err := cursor.Decode(&raw); err != nil {
			log.Printf("[MONGODB] Erro ao decodificar grupo: %v", err)
			continue
		}
		groups = append(groups, rawToTestGroup(&raw))
	}
	return groups, nil
}

// GetGroupByID busca um grupo pelo seu ID
func (m *MongoDBClient) GetGroupByID(ctx context.Context, idStr string) (*models.TestGroup, error) {
	if m.GroupsCollection == nil {
		return nil, fmt.Errorf("colecao de grupos nao inicializada")
	}

	var raw rawTestGroup
	err := m.GroupsCollection.FindOne(ctx, groupIDFilter(idStr)).Decode(&raw)
	if err != nil {
		return nil, fmt.Errorf("grupo nao encontrado: %w", err)
	}
	grp := rawToTestGroup(&raw)
	return &grp, nil
}

// CreateGroup cria um novo grupo de testes
func (m *MongoDBClient) CreateGroup(ctx context.Context, group *models.TestGroup) (string, error) {
	if m.GroupsCollection == nil {
		return "", fmt.Errorf("colecao de grupos nao inicializada")
	}

	if group.Name == "" {
		return "", fmt.Errorf("nome do grupo e obrigatorio")
	}
	if group.VLANs == nil {
		group.VLANs = []int{}
	}
	if group.IntervalMinutes <= 0 {
		group.IntervalMinutes = 60
	}

	now := time.Now().UTC()
	group.CreatedAt = now
	group.UpdatedAt = now

	// Se marcado como padrão, desmarca outros
	if group.IsDefault {
		_, _ = m.GroupsCollection.UpdateMany(ctx, bson.M{}, bson.M{"$set": bson.M{"is_default": false}})
	}

	newID := fmt.Sprintf("grp-%d", time.Now().UnixNano())
	group.ID = newID

	doc := bson.M{
		"_id":              newID,
		"name":             group.Name,
		"description":      group.Description,
		"vlans":            group.VLANs,
		"is_default":       group.IsDefault,
		"interval_minutes": group.IntervalMinutes,
		"modules":          group.Modules,
		"created_at":       group.CreatedAt,
		"updated_at":       group.UpdatedAt,
	}

	_, err := m.GroupsCollection.InsertOne(ctx, doc)
	if err != nil {
		return "", fmt.Errorf("erro ao inserir grupo no MongoDB: %w", err)
	}

	log.Printf("[MONGODB] Grupo '%s' criado com sucesso (ID: %s, VLANs: %v)", group.Name, newID, group.VLANs)
	return newID, nil
}

// UpdateGroup atualiza dados e módulos de um grupo existente
func (m *MongoDBClient) UpdateGroup(ctx context.Context, idStr string, group *models.TestGroup) error {
	if m.GroupsCollection == nil {
		return fmt.Errorf("colecao de grupos nao inicializada")
	}

	if group.Name == "" {
		return fmt.Errorf("nome do grupo e obrigatorio")
	}
	if group.VLANs == nil {
		group.VLANs = []int{}
	}
	if group.IntervalMinutes <= 0 {
		group.IntervalMinutes = 60
	}

	// Se marcado como padrão, desmarca outros
	if group.IsDefault {
		_, _ = m.GroupsCollection.UpdateMany(ctx, bson.M{}, bson.M{"$set": bson.M{"is_default": false}})
	}

	update := bson.M{
		"$set": bson.M{
			"name":             group.Name,
			"description":      group.Description,
			"vlans":            group.VLANs,
			"is_default":       group.IsDefault,
			"interval_minutes": group.IntervalMinutes,
			"modules":          group.Modules,
			"updated_at":       time.Now().UTC(),
		},
	}

	res, err := m.GroupsCollection.UpdateOne(ctx, groupIDFilter(idStr), update)
	if err != nil {
		return fmt.Errorf("erro ao atualizar grupo: %w", err)
	}
	if res.MatchedCount == 0 {
		return fmt.Errorf("grupo nao encontrado")
	}

	log.Printf("[MONGODB] Grupo '%s' (ID: %s) atualizado com sucesso", group.Name, idStr)
	return nil
}

// DeleteGroup remove um grupo existente
func (m *MongoDBClient) DeleteGroup(ctx context.Context, idStr string) error {
	if m.GroupsCollection == nil {
		return fmt.Errorf("colecao de grupos nao inicializada")
	}

	// Impede exclusão se for o único grupo restante ou grupo padrão sem substituto
	count, _ := m.GroupsCollection.CountDocuments(ctx, bson.M{})
	if count <= 1 {
		return fmt.Errorf("nao e possivel excluir o unico grupo existente do sistema")
	}

	res, err := m.GroupsCollection.DeleteOne(ctx, groupIDFilter(idStr))
	if err != nil {
		return fmt.Errorf("erro ao excluir grupo: %w", err)
	}
	if res.DeletedCount == 0 {
		return fmt.Errorf("grupo nao encontrado")
	}

	log.Printf("[MONGODB] Grupo ID '%s' excluido do sistema", idStr)
	return nil
}

// Disconnect encerra a conexão com o MongoDB
func (m *MongoDBClient) Disconnect(ctx context.Context) error {
	if m.Client != nil {
		return m.Client.Disconnect(ctx)
	}
	return nil
}

