package foundation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/weitecit/pkg/log"
	"github.com/weitecit/pkg/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var backupHistory = []string{}

type MongoRepository struct {
	Error               error
	ConnectionString    string
	RepoID              string
	DataBase            string
	Collection          string
	clientInstance      *mongo.Client
	clientInstanceError error
	mongoOnce           sync.Once
	ctx                 context.Context
}

func (m *MongoRepository) ToJSON() string {
	o, err := json.MarshalIndent(m, "", "\t")
	if err != nil {
		log.Err(err)
		return "Error in conversion"
	}
	return string(o)
}

func NewMongoRepository(connection string, repoID string, collection string, isGlobal bool) MongoRepository {
	m := &MongoRepository{}
	m.ConnectionString = connection
	m.RepoID = repoID
	m.DataBase = repoID
	m.Collection = collection
	m.ctx = context.Background()

	// Si el usuario no tienen una servidor de base de datos propio o es algo global
	// Abre la conexión de ASD
	if m.ConnectionString == "" || isGlobal {
		m.ConnectionString = utils.GetEnv("MONGO_REPO")
	}

	if isGlobal {
		m.DataBase = utils.GetEnv("DEFAULT_DATABASE")
	}

	if m.DataBase == "" {
		m.Error = errors.New("MongoRepository.NewMongoRepository: Database can not be empty")
	}

	return *m
}

func (m *MongoRepository) GetDB() (*mongo.Database, error) {

	response := &mongo.Database{}

	if m.ConnectionString == "" {
		err := errors.New("MongoRepository.GetDB: connection string can not be empty")
		log.Err(err)
		return response, err
	}

	if m.DataBase == "" {
		err := errors.New("MongoRepository.GetDB: not database name")
		log.Err(err)
		return response, err
	}

	m.GetMongoClient()
	if m.clientInstanceError != nil {
		if m.clientInstanceError.Error() == "error parsing uri: scheme must be \"mongodb\" or \"mongodb+srv\"" {
			m.clientInstanceError = errors.New("MongoRepository.GetDB: connection string is not valid: " + m.ConnectionString)
		}
		log.Err(m.clientInstanceError)
		return response, m.clientInstanceError

	}
	return m.clientInstance.Database(m.DataBase), nil
}

func (m *MongoRepository) GetMongoClient() {
	//Perform connection creation operation only once.

	for _, connection := range ConnectionPool {
		con := connection.(*MongoRepository)
		if con.ConnectionString != m.ConnectionString {
			continue
		}
		if con.DataBase != m.DataBase {
			continue
		}

		err := con.clientInstance.Ping(context.TODO(), nil)
		if err != nil {
			m.clientInstanceError = err
		}

		m.clientInstance = con.clientInstance
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	// Set client options
	clientOptions := options.Client().ApplyURI(m.ConnectionString + "/" + m.DataBase)
	// Connect to MongoDB
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		m.clientInstanceError = err
	}
	if client == nil {
		return
	}
	// Check the connection
	err = client.Ping(context.TODO(), nil)
	if err != nil {
		m.clientInstanceError = err
	}
	m.clientInstance = client

	ConnectionPool = append(ConnectionPool, m)

}

func (m *MongoRepository) Update(request RepoRequest) RepoResponse {
	response := &RepoResponse{}

	if request.Model == nil {
		response.Error = errors.New("MongoRepository.Update: model can not be empty")
		return *response
	}

	request.Model.SetUpdated(request.User)
	if request.Model.IsNew() {
		return m.create(request)
	}

	collection, err := m.GetCollection()
	if err != nil {
		log.Err(err)
		return m.create(request)
	}

	isNew := request.Model.IsNew()

	id, err := request.Model.GetID()
	if err != nil || isNew {
		log.Err(err)
		return m.create(request)
	}

	result, err := collection.UpdateOne(m.ctx, bson.M{"_id": id}, bson.M{"$set": request.Model})
	if err != nil {
		log.Err(err)
		return m.create(request)
	}

	response.TotalRows = result.ModifiedCount
	response.List = []interface{}{request.Model}

	return *response
}

func (m *MongoRepository) UpdateMany(request RepoRequest, values map[string]interface{}) RepoResponse {

	collection, err := m.GetCollection()
	if err != nil {
		log.Err(err)
		return RepoResponse{Error: err}
	}

	findOptions := request.FindOptions

	isEmpty := findOptions.filterIsEmpty()
	if isEmpty {
		err := errors.New("MongoRepository.UpdateMany: " + collection.Name() + " model can not be empty. Filter is empty")
		log.Trace(err)
		return RepoResponse{Error: err}
	}

	values["updated_by"] = request.User.GetUserLog()

	getFilter, err := m.GetFilter(findOptions)
	if err != nil {
		log.Err(err)
		return RepoResponse{Error: err}
	}

	response, err := collection.UpdateMany(m.ctx, getFilter, bson.M{"$set": values})
	if err != nil {
		log.Err(err)
		return RepoResponse{TotalRows: response.ModifiedCount, Error: err}
	}

	return RepoResponse{TotalRows: response.ModifiedCount}

}

func (m *MongoRepository) AddItemInArray(request RepoRequest, field string, value string) RepoResponse {
	id, err := utils.GetObjectIdFromString(request.ID)
	if err != nil {
		log.Err(err)
		return RepoResponse{Error: err}
	}

	collection, err := m.GetCollection()
	if err != nil {
		log.Err(err)
		return RepoResponse{Error: err}
	}

	// Pipeline para agregar el elemento si no existe
	updatePipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"_id": id}}},
		{{Key: "$addFields", Value: bson.M{
			field: bson.M{
				"$cond": bson.M{
					"if": bson.M{"$eq": []interface{}{bson.M{"$type": "$" + field}, "array"}},
					"then": bson.M{
						"$cond": bson.M{
							"if":   bson.M{"$in": []interface{}{value, "$" + field}},
							"then": "$" + field,
							"else": bson.M{"$concatArrays": []interface{}{"$" + field, []string{value}}},
						},
					},
					"else": []string{value},
				},
			},
		}}},
		{{Key: "$merge", Value: bson.M{
			"into":           collection.Name(),
			"on":             "_id",
			"whenMatched":    "merge",
			"whenNotMatched": "discard",
		}}},
	}

	if _, err := collection.Aggregate(context.TODO(), updatePipeline); err != nil {
		log.Err(err)
		return RepoResponse{Error: err}
	}

	return RepoResponse{TotalRows: 1} // Retorna un valor indicando éxito
}

func (m *MongoRepository) RemoveItemInArray(request RepoRequest, field string, value string) RepoResponse {
	id, err := utils.GetObjectIdFromString(request.ID)
	if err != nil {
		log.Err(err)
		return RepoResponse{Error: err}
	}

	collection, err := m.GetCollection()
	if err != nil {
		log.Err(err)
		return RepoResponse{Error: err}
	}

	// Pipeline para eliminar el elemento si existe
	updatePipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"_id": id}}},
		{{Key: "$addFields", Value: bson.M{
			field: bson.M{
				"$cond": bson.M{
					"if": bson.M{"$eq": []interface{}{bson.M{"$type": "$" + field}, "array"}},
					"then": bson.M{
						"$setDifference": []interface{}{"$" + field, []string{value}},
					},
					"else": []string{}, // Si no es un array, lo deja vacío
				},
			},
		}}},
		{{Key: "$merge", Value: bson.M{
			"into":           collection.Name(),
			"on":             "_id",
			"whenMatched":    "merge",
			"whenNotMatched": "discard",
		}}},
	}

	if _, err := collection.Aggregate(context.TODO(), updatePipeline); err != nil {
		log.Err(err)
		return RepoResponse{Error: err}
	}

	return RepoResponse{TotalRows: 1} // Retorna un valor indicando éxito
}

func (m *MongoRepository) SwitchItemInArray(request RepoRequest, field string, value string) RepoResponse {

	id, err := utils.GetObjectIdFromString(request.ID)
	if err != nil {
		log.Err(err)
		return RepoResponse{Error: err}
	}

	collection, err := m.GetCollection()
	if err != nil {
		log.Err(err)
		return RepoResponse{Error: err}
	}

	// First pipeline to update the array
	updatePipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"_id": id}}},
		{{Key: "$addFields", Value: bson.M{
			field: bson.M{
				"$cond": bson.M{
					"if": bson.M{"$eq": []interface{}{bson.M{"$type": "$" + field}, "array"}},
					"then": bson.M{
						"$cond": bson.M{
							"if":   bson.M{"$in": []interface{}{value, "$" + field}},
							"then": bson.M{"$setDifference": []interface{}{"$" + field, []string{value}}},
							"else": bson.M{"$concatArrays": []interface{}{"$" + field, []string{value}}},
						},
					},
					"else": []string{value},
				},
			},
		}}},
		{{Key: "$merge", Value: bson.M{
			"into":           collection.Name(),
			"on":             "_id",
			"whenMatched":    "merge",
			"whenNotMatched": "discard",
		}}},
	}

	_, err = collection.Aggregate(context.TODO(), updatePipeline)
	if err != nil {
		log.Err(err)
		return RepoResponse{Error: err}
	}

	// Second query to get the array count
	var result struct {
		Count int `bson:"count"`
	}

	countPipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"_id": id}}},
		{{Key: "$project", Value: bson.M{
			"count": bson.M{
				"$cond": bson.M{
					"if":   bson.M{"$eq": []interface{}{bson.M{"$type": "$" + field}, "array"}},
					"then": bson.M{"$size": "$" + field},
					"else": 0,
				},
			},
		}}},
	}

	cursor, err := collection.Aggregate(context.TODO(), countPipeline)
	if err != nil {
		log.Err(err)
		return RepoResponse{Error: err}
	}
	defer cursor.Close(context.TODO())

	if cursor.Next(context.TODO()) {
		if err := cursor.Decode(&result); err != nil {
			log.Err(err)
			return RepoResponse{Error: err}
		}
	}

	return RepoResponse{TotalRows: int64(result.Count)}
}

func (m *MongoRepository) UpdateField(request RepoRequest, field string, value interface{}) RepoResponse {

	collection, err := m.GetCollection()
	if err != nil {
		log.Err(err)
		return RepoResponse{Error: err}
	}

	findOptions := request.FindOptions

	isEmpty := findOptions.filterIsEmpty()
	if isEmpty {
		err := errors.New("MongoRepository.UpdateField: " + collection.Name() + " model can not be empty. Filter is empty")
		log.Trace(err)
		return RepoResponse{Error: err}
	}

	values := map[string]interface{}{
		field: value,
	}

	getFilter, err := m.GetFilter(findOptions)
	if err != nil {
		log.Err(err)
		return RepoResponse{Error: err}
	}

	response, err := collection.UpdateMany(m.ctx, getFilter, bson.M{"$set": values})
	if err != nil {
		log.Err(err)
		return RepoResponse{TotalRows: response.ModifiedCount, Error: err}
	}

	return RepoResponse{TotalRows: response.ModifiedCount}

}

func (m *MongoRepository) Move(request RepoRequest) RepoResponse {

	collection, err := m.GetCollection()
	if err != nil {
		log.Err(err)
		return RepoResponse{Error: err}
	}

	if request.TargetCollection == "" {
		err := errors.New("MongoRepository.Move: move collection can not be empty")
		log.Err(err)
		return RepoResponse{Error: err}
	}

	findOptions := request.FindOptions

	isEmpty := findOptions.filterIsEmpty()
	if isEmpty {
		err := errors.New("MongoRepository.Move: " + collection.Name() + " model can not be empty. Filter is empty")
		log.Trace(err)
		return RepoResponse{Error: err}
	}

	getFilter, err := m.GetFilter(findOptions)
	if err != nil {
		log.Err(err)
		return RepoResponse{Error: err}
	}

	// out aggregate
	_, err = collection.Aggregate(context.TODO(), mongo.Pipeline{
		{{Key: "$match", Value: getFilter}},
		// {{Key: "$out", Value: request.TargetCollection}},
		{{Key: "$merge", Value: request.TargetCollection}},
	})
	if err != nil {
		log.Err(err)
		return RepoResponse{TotalRows: -1, Error: err}
	}

	result, err := collection.DeleteMany(m.ctx, getFilter)
	if err != nil {

		log.Err(err)
		return RepoResponse{Error: err}
	}

	return RepoResponse{TotalRows: result.DeletedCount}

}

func (m *MongoRepository) DeleteSoft(request RepoRequest) RepoResponse {

	collection, err := m.GetCollection()
	if err != nil {
		log.Err(err)
		return RepoResponse{Error: err}
	}

	// Soft delete can avoid filterIsEmpty check
	userLog := request.User.GetUserLog()

	values := map[string]interface{}{
		"deleted_by": userLog,
	}

	getFilter, err := m.GetFilter(request.FindOptions)
	if err != nil {
		log.Err(err)
		return RepoResponse{Error: err}
	}

	response, err := collection.UpdateMany(m.ctx, getFilter, bson.M{"$set": values})
	if err != nil {
		log.Err(err)
		return RepoResponse{TotalRows: response.ModifiedCount, Error: err}
	}

	return RepoResponse{TotalRows: response.ModifiedCount}
}

func (m *MongoRepository) RemoveField(request RepoRequest, field string) RepoResponse {

	collection, err := m.GetCollection()
	if err != nil {
		log.Err(err)
		return RepoResponse{Error: err}
	}

	// Soft delete can avoid filterIsEmpty check
	userLog := request.User.GetUserLog()

	values := map[string]interface{}{
		"updated_by": userLog,
		field:        nil,
	}

	getFilter, err := m.GetFilter(request.FindOptions)
	if err != nil {
		log.Err(err)
		return RepoResponse{Error: err}
	}

	response, err := collection.UpdateMany(m.ctx, getFilter, bson.M{"$set": values})
	if err != nil {
		log.Err(err)
		return RepoResponse{TotalRows: response.ModifiedCount, Error: err}
	}

	return RepoResponse{TotalRows: response.ModifiedCount}
}

func (m *MongoRepository) create(request RepoRequest) RepoResponse {

	model := request.Model
	model.SetCreated(request.User)

	collection, err := m.GetCollection()
	if err != nil {
		log.Err(err)
		return RepoResponse{Error: err}
	}

	_, err = collection.InsertOne(m.ctx, model)
	if err != nil {
		log.Err(err)
		return RepoResponse{Error: err}
	}

	return RepoResponse{
		TotalRows: 1,
		List:      []interface{}{model},
	}
}

func (m *MongoRepository) CreateMany(dbModel RepositoryModel, list []interface{}) error {
	if len(list) == 0 {
		return nil
	}

	collection, err := m.GetCollection()
	if err != nil {
		log.Err(err)
		return err
	}

	_, err = collection.InsertMany(m.ctx, list)
	if err != nil {
		log.Err(err)
		return err
	}
	return nil
}

func (m *MongoRepository) FindOne(request RepoRequest) RepoResponse {
	response := &RepoResponse{}

	if request.Model == nil {
		response.Error = errors.New("MongoRepository.FindOne: model can not be empty")
		return *response
	}

	collection, err := m.GetCollection()
	if err != nil {
		log.Err(err)
		response.Error = err
		return *response
	}

	id, err := request.Model.GetID()
	if err != nil {
		log.Err(err)
		response.Error = err
		return *response
	}

	result := collection.FindOne(m.ctx, bson.M{"_id": id})
	err = result.Err()
	if err != nil {
		if err.Error() == "mongo: no documents in result" {
			message := fmt.Sprintf("MongoRepository.GetDetail.%s: no document found, ID: %s", collection.Name(), id)
			err = errors.New(message)
		}
		log.Trace(err)
		response.Error = err
		return *response
	}

	response.TotalRows = 1
	response.Error = result.Decode(request.Model)

	return *response
}

func (m *MongoRepository) Find(request RepoRequest) RepoResponse {

	_, err := request.Model.GetID()
	if err == nil {
		result := m.FindOne(request)
		response := &RepoResponse{}
		response.CurrentPage = result.CurrentPage
		response.Error = result.Error
		if response.Error != nil && strings.Contains(response.Error.Error(), "no document found") {
			response.Error = nil
		}
		response.List = []interface{}{request.Model}
		response.PageSize = result.PageSize
		response.TotalPages = result.TotalPages
		response.TotalRows = result.TotalRows
		return *response
	}

	response := &RepoResponse{
		List: request.List,
	}
	collection, err := m.GetCollection()
	if err != nil {
		log.Err(err)
		response.Error = err
		return *response
	}

	countOptions := options.Count()
	countOptions.Limit = utils.Int64(1000001)

	options := options.Find()

	if request.FindOptions.GetTotalOrders() > 0 {
		options.SetSort(m.GetOrder(request.FindOptions))
	}

	skippedRows := request.PageSize * (request.CurrentPage - 1)
	if skippedRows < 0 {
		skippedRows = 0
	}

	if request.PageSize > 0 && skippedRows > 0 {
		options.Skip = &skippedRows
	}

	if request.CurrentPage > 0 {
		options.SetLimit(request.PageSize)
	}

	filter, err := m.GetFilter(request.FindOptions)
	if err != nil {
		log.Err(err)
		response.Error = err
		return *response
	}

	cursor, err := collection.Find(m.ctx, filter, options)

	if err != nil {
		log.Trace(err)
		response.Error = err
		return *response
	}

	count, err := collection.CountDocuments(m.ctx, filter, countOptions)
	if err != nil {
		log.Trace(err)
		return *response
	}
	response.TotalRows = count

	err = cursor.All(m.ctx, &response.List)
	if err != nil {
		log.Err(err)
		response.Error = err
		return *response
	}

	response.CurrentPage = request.CurrentPage
	response.PageSize = request.PageSize

	if count > 0 && request.PageSize > 0 {
		response.TotalPages = count / request.PageSize
		if count/request.PageSize > 0 {
			response.TotalPages++
		}
	}

	return *response
}

func (m *MongoRepository) Count(request RepoRequest) RepoResponse {
	findResponse := &RepoResponse{}
	collection, err := m.GetCollection()
	if err != nil {
		log.Err(err)
		findResponse.Error = err
		return *findResponse
	}

	countOptions := options.Count()
	if request.PageSize > 0 {
		countOptions.Limit = utils.Int64(request.PageSize)
	}

	getFilter, err := m.GetFilter(request.FindOptions)
	if err != nil {
		log.Err(err)
		findResponse.Error = err
		return *findResponse
	}

	count, err := collection.CountDocuments(m.ctx, getFilter, countOptions)
	if err != nil {
		log.Err(err)
		return *findResponse
	}
	findResponse.TotalRows = count

	return *findResponse
}

func (m *MongoRepository) Delete(request RepoRequest) RepoResponse {
	collection, err := m.GetCollection()
	if err != nil {
		log.Err(err)
		return RepoResponse{Error: err}
	}

	id, err := request.Model.GetID()
	if err != nil {
		if err.Error() != "BaseModel.GetID: ID is nil" {
			log.Trace(err)
			return RepoResponse{Error: err}
		}
	}

	if id != nil {
		_, err := collection.DeleteOne(m.ctx, bson.M{"_id": id})
		if err != nil {
			log.Trace(err)
			return RepoResponse{Error: err}
		}
		return RepoResponse{TotalRows: 1}
	}

	isEmpty := request.FindOptions.filterIsEmpty()
	if isEmpty {
		err := errors.New("MongoRepository.Delete: " + collection.Name() + " model can not be empty")
		log.Trace(err)
		return RepoResponse{Error: err}
	}

	getFilter, err := m.GetFilter(request.FindOptions)
	if err != nil {
		log.Err(err)
		return RepoResponse{Error: err}
	}

	result, err := collection.DeleteMany(m.ctx, getFilter)
	if err != nil {
		log.Err(err)
		return RepoResponse{Error: err}
	}

	return RepoResponse{TotalRows: result.DeletedCount}
}

func (m *MongoRepository) DeleteAll(dbModel RepositoryModel) error {
	collection, err := m.GetCollection()
	if err != nil {
		log.Err(err)
		return err
	}

	_, err = collection.DeleteMany(m.ctx, dbModel)
	if err != nil {
		log.Err(err)
	}

	return err
}
func (m *MongoRepository) GetSize(dbModel RepositoryModel) error {
	collection, err := m.GetCollection()
	if err != nil {
		log.Err(err)
		return err
	}

	db, err := m.GetDB()
	if err != nil {
		log.Err(err)
		return err
	}

	result := db.RunCommand(context.Background(), bson.M{"collStats": collection.Name()})

	var document bson.M
	err = result.Decode(&document)

	if err != nil {
		panic(err)
	}

	fmt.Printf("Collection size: %v Bytes\n", document["size"])
	fmt.Printf("Average object size: %v Bytes\n", document["avgObjSize"])
	fmt.Printf("Storage size: %v Bytes\n", document["storageSize"])
	fmt.Printf("Total index size: %v Bytes\n", document["totalIndexSize"])

	return nil
}

func (m *MongoRepository) Aggregate(request RepoRequest) RepoResponse {
	response := &RepoResponse{
		List: request.List,
	}

	if request.Pipeline == nil {
		err := errors.New("MongoRepository.Aggregate: Pipeline is nil")
		log.Trace(err)
		response.Error = err
		return *response
	}
	// pipeline := map[string]interface{}{}
	// pipeline := bson.A{}
	pipeline := request.Pipeline.(bson.A)

	collection, err := m.GetCollection()
	if err != nil {
		log.Err(err)
		response.Error = err
		return *response
	}

	countOptions := options.Count()
	countOptions.Limit = utils.Int64(1000001)

	aggregateOptions := options.Aggregate()

	cursor, err := collection.Aggregate(m.ctx, pipeline, aggregateOptions)
	if err != nil {
		log.Err(err)
		response.Error = err
		return *response
	}

	err = cursor.All(m.ctx, &response.List)
	if err != nil {
		log.Err(err)
		response.Error = err
		return *response
	}

	pipeline = append(pipeline, bson.D{{Key: "$count", Value: "total"}})

	cursor, err = collection.Aggregate(m.ctx, pipeline, aggregateOptions)
	if err != nil {
		response.Error = err

		return *response
	}

	type Counter struct {
		Total int64 `json:"total" bson:"total"`
	}

	var counter []Counter
	err = cursor.All(m.ctx, &counter)
	if err != nil {
		response.Error = err
		return *response
	}

	if len(counter) == 0 {
		return *response
	}

	response.TotalRows = counter[0].Total

	return *response
}

func (m *MongoRepository) DeleteDataBasesByCollections(list []utils.Dictionary, exceptions []string) error {
	for _, item := range list {
		m.ConnectionString = item.Key
		m.DataBase = item.Value
		db, err := m.GetDB()
		if err != nil {
			log.Err(err)
			return err
		}
		collections, err := db.ListCollectionNames(m.ctx, bson.M{})
		if err != nil {
			log.Err(err)
			return err
		}
		for _, col := range collections {
			matched := false
			for _, item := range exceptions {
				if item == col {
					matched = true
				}
			}
			if matched {
				continue
			}
			db, err := m.GetDB()
			if err != nil {
				log.Err(err)
				return err
			}
			_, err = db.Collection(col).DeleteMany(m.ctx, bson.M{})
		}
	}
	return nil
}

func (m *MongoRepository) DeleteDatabase(connection string, database string) error {

	if m.ConnectionString == "" || connection != "" {
		m.ConnectionString = connection
	}
	if m.DataBase == "" || database != "" {
		m.DataBase = database
	}

	if m.ConnectionString == "" || m.DataBase == "" {
		err := errors.New("MongoRepository.DeleteDatabase: connectionString or DataBase can not be empty")
		log.Err(err)
		return err
	}

	db, err := m.GetDB()
	if err != nil {
		log.Err(err)
		return err
	}
	return db.Drop(m.ctx)

}

func (m *MongoRepository) GetCollection() (*mongo.Collection, error) {

	if m.Collection == "" {
		err := errors.New("MongoRepository.GetCollection: not collection assigned: " + m.Collection)
		return nil, err
	}

	db, err := m.GetDB()
	if err != nil {
		log.Err(err)
		return nil, err
	}
	return db.Collection(m.Collection), nil
}

func (m *MongoRepository) GetFilter(filterOptions FindOptions) (map[string]interface{}, error) {
	result := bson.M{}
	andFilters := []bson.M{}

	for _, filter := range filterOptions.Filters {
		// Detect if the value is a date string and convert it
		if strVal, ok := filter.Value.(string); ok {
			// Try to parse common date formats
			var parsedTime time.Time
			var err error

			// Try RFC3339 format first
			parsedTime, err = time.Parse(time.RFC3339, strVal)
			if err != nil {
				// Try ISO format
				parsedTime, err = time.Parse("2006-01-02", strVal)
				if err != nil {
					// Try ISO format with time
					parsedTime, err = time.Parse("2006-01-02T15:04:05", strVal)
					if err != nil {
						// Try ISO format with timezone
						parsedTime, err = time.Parse("2006-01-02T15:04:05-07:00", strVal)
						if err != nil {
							// If none of the formats work, keep it as string
							goto skipDateConversion
						}
					}
				}
			}
			filter.Value = parsedTime
		}
	skipDateConversion:

		filterItem, err := m.getFilterItem(filter)
		if err != nil {
			return map[string]interface{}{}, err
		}

		// Store filter item
		andFilters = append(andFilters, bson.M{filter.Key: filterItem})
	}

	// If we have any andFilters, combine them with the main result
	if len(andFilters) > 0 {
		result["$and"] = andFilters
	}

	return result, nil
}

func (m *MongoRepository) getFilterItem(filter Filter) (interface{}, error) {
	switch filter.Operator {
	case FilterOperatorEquals:
		return filter.Value, nil
	case FilterOperatorEqualsWithCaseInsensitive:
		return primitive.Regex{Pattern: "^" + filter.Value.(string) + "$", Options: "i"}, nil
	case FilterOperatorNotEquals:
		return bson.M{"$ne": filter.Value}, nil
	case FilterOperatorIn:
		return bson.M{"$in": filter.Value}, nil
	case FilterOperatorNotIn:
		return bson.M{"$nin": filter.Value}, nil
	case FilterOperatorSize:
		return bson.M{"$size": filter.Value}, nil
	case FilterOperatorAll:
		return bson.M{"$all": filter.Value}, nil
	case FilterOperatorContains:
		return primitive.Regex{Pattern: filter.Value.(string), Options: "i"}, nil
	case FilterOperatorGroupsOfArrays:
		return bson.M{"$elemMatch": bson.M{"$not": bson.M{"$elemMatch": bson.M{"$nin": filter.Value}}}}, nil
	case FilterOperatorGreat:
		return bson.M{"$gt": filter.Value}, nil
	case FilterOperatorLess:
		return bson.M{"$lt": filter.Value}, nil
	case FilterOperatorGreatOrEqual:
		return bson.M{"$gte": filter.Value}, nil
	case FilterOperatorLessOrEqual:
		return bson.M{"$lte": filter.Value}, nil
	case FilterOperatorNotNil:
		return bson.M{"$exists": true}, nil
	case FilterOperatorNil:
		return bson.M{"$exists": false}, nil
	default:
		return bson.D{}, errors.New("MongoRepository.getFilterItem: unknown filter operator: ")
	}
}

func (m *MongoRepository) GetOrder(filterOptions FindOptions) map[string]interface{} {
	// if m.ConnectionString == "" {
	// 	err := errors.New("MongoRepository.GetDB: connection string can not be empty")
	println("•••••••••••••••••••••••••••••••••")
	println("GetOrder Not implemented")
	println("•••••••••••••••••••••••••••••••••")
	return bson.M{}

}

func (m *MongoRepository) GetType() RepoType {
	return RepoTypeMongoDB
}

func (m *MongoRepository) GetRepoID() string {
	return m.RepoID
}

func (m *MongoRepository) GetDataBase() string {
	return m.DataBase
}

func (m *MongoRepository) GetConnection() string {
	return m.ConnectionString
}

func (m *MongoRepository) SetRepoID(repoID string) error {

	if utils.IsEmptyStr(repoID) {
		return errors.New("MongoRepository.SetRepoID: repoID can not be empty")
	}
	m.RepoID = repoID
	return nil
}

func (m *MongoRepository) RepoBackup(request RepoRequest, backupID string) RepoResponse {
	out := "backup/" + m.DataBase + "/" + backupID
	db := m.DataBase

	cmd := exec.Command("mongodump", "--db", db, "--out", out)
	err := cmd.Run()
	if err != nil {
		return RepoResponse{Error: err}
	}

	return RepoResponse{Error: nil}
}

func (m *MongoRepository) RepoRestore(request RepoRequest, backupID string) RepoResponse {
	db := m.DataBase

	err := m.DeleteDatabase(m.GetConnection(), m.DataBase)
	if err != nil {
		return RepoResponse{Error: err}
	}
	out := "backup/" + db + "/" + backupID + "/" + db

	cmd := exec.Command("mongorestore", "--db", db, out)
	err = cmd.Run()
	if err != nil {
		return RepoResponse{Error: err}
	}

	return RepoResponse{Error: nil}
}
