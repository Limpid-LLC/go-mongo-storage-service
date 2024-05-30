package mongo

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/webmakom-com/saiStorage/config"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Client struct {
	Config config.Configuration
	Host   *mongo.Client
	Ctx    context.Context
}

type FindResult struct {
	Count  int64         `json:"count,omitempty"`
	Result []interface{} `json:"result,omitempty"`
}

type Options struct {
	NoSet bool   `json:"noset"`
	Limit int64  `json:"limit"`
	Skip  int64  `json:"skip"`
	Sort  bson.M `json:"sort"`
	Count int64  `json:"count"`
	Batch int64  `json:"batch"`
}

func NewMongoClient(config config.Configuration) (Client, error) {
	var host *mongo.Client
	var hostErr error

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if config.Storage.ConnectionString != "" {
		host, _ = mongo.NewClient(options.Client().ApplyURI(config.Storage.ConnectionString))

		hostErr = host.Connect(ctx)
	} else {
		switch config.Storage.Atlas {
		case false:
			{
				host, _ = mongo.NewClient(options.Client().ApplyURI(
					"mongodb://" + config.Storage.Host + ":" + config.Storage.Port + "/" + config.Storage.Database,
				))

				hostErr = host.Connect(ctx)
			}
		default:
			{
				host, hostErr = mongo.Connect(ctx, options.Client().ApplyURI(
					"mongodb+srv://"+config.Storage.User+":"+config.Storage.Pass+"@"+config.Storage.Host+"/"+config.Storage.Database+"?ssl=true&authSource=admin&retryWrites=true&w=majority",
				))
			}
		}
	}

	client := Client{
		Ctx:    ctx,
		Config: config,
		Host:   host,
	}

	if hostErr != nil {
		return client, hostErr
	}

	return client, nil
}

func (c Client) GetCollection(collectionName string) *mongo.Collection {
	return c.Host.Database(c.Config.Storage.Database).Collection(collectionName)
}

func (c Client) FindOne(collectionName string, selector map[string]interface{}) (map[string]interface{}, error) {
	var result map[string]interface{}
	collection := c.GetCollection(collectionName)
	selector = c.preprocessSelector(selector)
	cur, err := collection.Find(context.TODO(), selector)

	if err != nil {
		return result, err
	}

	defer cur.Close(context.TODO())

	for cur.Next(context.TODO()) {
		var elem map[string]interface{}
		decodeErr := cur.Decode(&elem)

		if decodeErr != nil {
			return result, decodeErr
		}

		result = elem
		break
	}

	if cursorErr := cur.Err(); cursorErr != nil {
		return result, cursorErr
	}

	return result, nil
}

func (c Client) Find(collectionName string, selector map[string]interface{}, inputOptions Options, includeFields []string) (*FindResult, error) {
	findResult := &FindResult{}
	requestOptions := options.Find()

	if inputOptions.Count != 0 {
		collection := c.GetCollection(collectionName)
		selector = c.preprocessSelector(selector)
		count, err := collection.CountDocuments(context.TODO(), selector)
		if err != nil {
			return &FindResult{}, err
		}
		return &FindResult{
			Count: count,
		}, nil
	}

	if inputOptions.Sort != nil {
		requestOptions.SetSort(inputOptions.Sort)
	}

	if inputOptions.Skip != 0 {
		requestOptions.SetSkip(inputOptions.Skip)
	}

	if inputOptions.Limit != 0 {
		requestOptions.SetLimit(inputOptions.Limit)
	}

	if inputOptions.Batch > 0 {
		requestOptions.SetAllowDiskUse(true)
		requestOptions.SetBatchSize(int32(inputOptions.Batch))

	}

	if includeFields != nil {
		projection := bson.D{}
		for _, v := range includeFields {
			projection = append(projection, bson.E{v, 1})
		}
		requestOptions.SetProjection(projection)
	}

	collection := c.GetCollection(collectionName)
	selector = c.preprocessSelector(selector)

	cur, err := collection.Find(context.TODO(), selector, requestOptions)

	if err != nil {
		return &FindResult{}, err
	}

	defer cur.Close(context.TODO())

	for cur.Next(context.TODO()) {
		var elem map[string]interface{}
		decodeErr := cur.Decode(&elem)

		if decodeErr != nil {
			return &FindResult{}, decodeErr
		}

		findResult.Result = append(findResult.Result, elem)
	}

	if cursorErr := cur.Err(); cursorErr != nil {
		return findResult, cursorErr
	}

	return findResult, nil
}

func (c Client) Insert(collectionName string, doc interface{}) (*mongo.InsertOneResult, error) {
	collection := c.GetCollection(collectionName)

	//processedDoc := c.preprocessDoc(doc)
	result, err := collection.InsertOne(context.TODO(), doc)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (c Client) Update(collectionName string, selector map[string]interface{}, update interface{}) (*mongo.UpdateResult, error) {
	collection := c.GetCollection(collectionName)
	selector = c.preprocessSelector(selector)

	result, err := collection.UpdateMany(context.TODO(), selector, update)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (c Client) Upsert(collectionName string, selector map[string]interface{}, update interface{}) error {
	collection := c.GetCollection(collectionName)
	requestOptions := options.Update().SetUpsert(true)
	selector = c.preprocessSelector(selector)

	_, err := collection.UpdateMany(context.TODO(), selector, update, requestOptions)
	if err != nil {
		return err
	}

	return nil
}

func (c Client) Remove(collectionName string, selector map[string]interface{}) error {
	collection := c.GetCollection(collectionName)
	selector = c.preprocessSelector(selector)

	_, err := collection.DeleteMany(context.TODO(), selector)
	if err != nil {
		return err
	}

	return nil
}

func (c Client) preprocessSelector(selector map[string]interface{}) map[string]interface{} {
	if selector["_id"] != nil {
		switch selector["_id"].(type) {
		case string:
			objID, err := primitive.ObjectIDFromHex(selector["_id"].(string))
			if err != nil {
				fmt.Println("Wrong objectId string")
				return selector
			}
			selector["_id"] = objID
		case map[string]interface{}:
			objIDslice := make([]primitive.ObjectID, 0)
			m := selector["_id"].(map[string]interface{})
			for k, v := range m {
				switch v.(type) {
				case []interface{}:
					for _, s := range v.([]interface{}) {
						objID, err := primitive.ObjectIDFromHex(s.(string))
						if err != nil {
							continue
						}
						objIDslice = append(objIDslice, objID)
					}
					m[k] = objIDslice
				default:
					fmt.Printf("wrong type for preprocessSelector: %+v, type : %s", m, reflect.TypeOf(v))
					return selector
				}

			}
			selector["_id"] = m
		default:
			fmt.Printf("wrong type for preprocessSelector: %+v, type : %s", selector["_id"], reflect.TypeOf(selector["_id"]))
			return selector
		}
	}
	return selector
}

// preprocess doc (insert method)
func (c Client) preprocessDoc(doc interface{}) primitive.M {
	m, ok := doc.(primitive.M)
	if !ok {
		return nil
	}
	if m["type"] == "refresh_token" {
		accessToken := m["access_token"].(map[string]interface{})
		id := accessToken["_id"].(string)
		objID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			fmt.Println("Preprocess doc - cant create objectID for access token")
			return nil
		}
		accessToken["_id"] = objID
		return m

	} else {
		return m
	}
}
