package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/webmakom-com/saiStorage/mongo"
	"github.com/webmakom-com/saiStorage/utils"
	"go.mongodb.org/mongo-driver/bson"
)

const (
	getMethod    = "get"
	saveMethod   = "save"
	updateMethod = "update"
	upsertMethod = "upsert"
	removeMethod = "remove"
)

func (s Server) handleWebSocketRequest(msg []byte) {

}

type jsonRequestType struct {
	Collection    string        `json:"collection"`
	Select        bson.M        `json:"select,omitempty"`
	Options       mongo.Options `json:"options"`
	Data          bson.M        `json:"data"`
	IncludeFields []string      `json:"include_fields,omitempty"`
	Method        string        `json:"method,omitempty"`
	Result        []interface{} `json:"result"`
}

type duplicatedRequest struct {
	Data   []byte `json:"data"`
	Method string `json:"method"`
}

func (s Server) handleServerRequest(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/get":
		{
			s.get(w, r, "get")
		}
	case "/save":
		{
			s.save(w, r, "save")
		}
	case "/update":
		{
			s.update(w, r, "update")
		}
	case "/upsert":
		{
			s.upsert(w, r, "upsert")
		}
	case "/remove":
		{
			s.remove(w, r, "remove")
		}
	}
}

func (s Server) get(w http.ResponseWriter, r *http.Request, method string) {
	var request jsonRequestType
	decoder := json.NewDecoder(r.Body)
	decoderErr := decoder.Decode(&request)

	if decoderErr != nil {
		log.Printf("Wrong JSON: %v", decoderErr)
		return
	}

	if s.Config.UsePermissionAuth {
		err := s.checkPermissionRequest(r, request.Collection, method, request.Select)
		if err != nil {
			log.Println(err)
			w.Write([]byte(err.Error()))
			return
		}
	}

	result, mongoErr := s.Client.Find(request.Collection, request.Select, request.Options, request.IncludeFields)

	if mongoErr != nil {
		log.Println("Mongo error:", mongoErr)
		return
	}

	request.Result = result.Result

	s.duplicateRequest(request, getMethod, s.Config.DuplicateMethod)

	_, writeErr := w.Write(utils.ConvertInterfaceToJson(result))

	if writeErr != nil {
		log.Println("Write error:", writeErr)
		return
	}
}

func (s Server) save(w http.ResponseWriter, r *http.Request, method string) {
	var request jsonRequestType
	decoder := json.NewDecoder(r.Body)
	decoderErr := decoder.Decode(&request)

	if decoderErr != nil {
		fmt.Printf("Wrong JSON: %v", decoderErr)
		return
	}

	if s.Config.UsePermissionAuth {
		err := s.checkPermissionRequest(r, request.Collection, method, request.Select)
		if err != nil {
			fmt.Println(err)
			w.Write([]byte(err.Error()))
			return
		}
	}

	id := uuid.New().String()
	request.Data["internal_id"] = id
	request.Data["cr_time"] = time.Now().Unix()
	request.Data["ch_time"] = time.Now().Unix()

	mongoErr := s.Client.Insert(request.Collection, request.Data)

	if mongoErr != nil {
		fmt.Println("Mongo error:", mongoErr)
		return
	}

	result, mongoGetErr := s.Client.Find(request.Collection, request.Data, request.Options, request.IncludeFields)

	if mongoGetErr != nil {
		log.Println("Mongo get error:", mongoGetErr)
		return
	}

	request.Result = result.Result

	s.duplicateRequest(request, saveMethod, s.Config.DuplicateMethod)

	_, writeErr := w.Write(utils.ConvertInterfaceToJson(bson.M{"Status": "Ok", "Result": id}))

	if writeErr != nil {
		fmt.Println("Write error:", writeErr)
		return
	}
}

func (s Server) update(w http.ResponseWriter, r *http.Request, method string) {
	var request jsonRequestType
	decoder := json.NewDecoder(r.Body)
	decoderErr := decoder.Decode(&request)

	if decoderErr != nil {
		fmt.Printf("Wrong JSON: %v", decoderErr)
		return
	}

	request.Data["ch_time"] = time.Now().Unix()

	if s.Config.UsePermissionAuth {
		err := s.checkPermissionRequest(r, request.Collection, method, request.Select)
		if err != nil {
			fmt.Println(err)
			w.Write([]byte(err.Error()))
			return
		}
	}

	mongoErr := s.Client.Update(request.Collection, request.Select, bson.M{"$set": request.Data})

	if mongoErr != nil {
		fmt.Println("Mongo error:", mongoErr)
		return
	}

	result, mongoGetErr := s.Client.Find(request.Collection, request.Select, request.Options, request.IncludeFields)

	if mongoGetErr != nil {
		log.Println("Mongo get error:", mongoGetErr)
		return
	}

	request.Result = result.Result

	s.duplicateRequest(request, updateMethod, s.Config.DuplicateMethod)

	_, writeErr := w.Write(utils.ConvertInterfaceToJson(bson.M{"Status": "Ok"}))

	if writeErr != nil {
		fmt.Println("Write error:", writeErr)
		return
	}
}

func (s Server) upsert(w http.ResponseWriter, r *http.Request, method string) {
	var request jsonRequestType
	decoder := json.NewDecoder(r.Body)
	decoderErr := decoder.Decode(&request)

	if decoderErr != nil {
		fmt.Printf("Wrong JSON: %v", decoderErr)
		return
	}

	if s.Config.UsePermissionAuth {
		err := s.checkPermissionRequest(r, request.Collection, method, request.Select)
		if err != nil {
			fmt.Println(err)
			w.Write([]byte(err.Error()))
			return
		}
	}

	result, mongoErr := s.Client.Find(request.Collection, request.Select, request.Options, request.IncludeFields)

	if mongoErr != nil {
		fmt.Println("Mongo error:", mongoErr)
		return
	}

	var id string
	var lastResult *mongo.FindResult

	if len(result.Result) > 0 {
		//update
		item, ok := result.Result[0].(map[string]interface{})
		if ok {
			id = item["internal_id"].(string)
		}
		request.Data["ch_time"] = time.Now().Unix()

		mongoErr := s.Client.Update(request.Collection, request.Select, bson.M{"$set": request.Data})
		if mongoErr != nil {
			fmt.Println("Mongo error:", mongoErr)
			return
		}

		lastResult, mongoErr = s.Client.Find(request.Collection, request.Select, request.Options, request.IncludeFields)
		if mongoErr != nil {
			log.Println("Mongo get error:", mongoErr)
			return
		}
	} else {
		//insert
		id = uuid.New().String()
		request.Data["internal_id"] = id
		request.Data["cr_time"] = time.Now().Unix()
		request.Data["ch_time"] = time.Now().Unix()

		mongoErr := s.Client.Insert(request.Collection, request.Data)
		if mongoErr != nil {
			fmt.Println("Mongo error:", mongoErr)
			return
		}

		lastResult, mongoErr = s.Client.Find(request.Collection, request.Data, request.Options, request.IncludeFields)
		if mongoErr != nil {
			log.Println("Mongo get error:", mongoErr)
			return
		}
	}

	request.Result = lastResult.Result

	s.duplicateRequest(request, upsertMethod, s.Config.DuplicateMethod)

	_, writeErr := w.Write(utils.ConvertInterfaceToJson(bson.M{"Status": "Ok", "Result": id}))

	if writeErr != nil {
		fmt.Println("Write error:", writeErr)
		return
	}
}

func (s Server) remove(w http.ResponseWriter, r *http.Request, method string) {
	var request jsonRequestType
	decoder := json.NewDecoder(r.Body)
	decoderErr := decoder.Decode(&request)

	if decoderErr != nil {
		fmt.Printf("Wrong JSON: %v", decoderErr)
		return
	}

	if s.Config.UsePermissionAuth {
		err := s.checkPermissionRequest(r, request.Collection, method, request.Select)
		if err != nil {
			fmt.Println(err)
			w.Write([]byte(err.Error()))
			return
		}
	}

	result, mongoErr := s.Client.Find(request.Collection, request.Select, request.Options, request.IncludeFields)

	if mongoErr != nil {
		fmt.Println("Mongo error:", mongoErr)
		return
	}

	mongoErr = s.Client.Remove(request.Collection, request.Select)

	if mongoErr != nil {
		fmt.Println("Mongo error:", mongoErr)
		return
	}

	request.Result = result.Result

	s.duplicateRequest(request, removeMethod, s.Config.DuplicateMethod)

	_, writeErr := w.Write(utils.ConvertInterfaceToJson(bson.M{"Status": "Ok"}))

	if writeErr != nil {
		fmt.Println("Write error:", writeErr)
		return
	}
}

func (s *Server) duplicateRequest(request jsonRequestType, storageMethod, handlerMethod string) {
	if s.Config.Duplication {
		go func() {
			request.Method = storageMethod
			b, err := json.Marshal(request)
			if err != nil {
				log.Printf("handler - %s - json.Marshal : %s", handlerMethod, err.Error())
				return
			}
			dupRequest := &duplicatedRequest{
				Data:   b,
				Method: handlerMethod,
			}

			data, err := json.Marshal(dupRequest)
			if err != nil {
				log.Printf("handler - %s - json.Marshal : %s", handlerMethod, err.Error())
				return
			}
			s.DuplicateCh <- bytes.NewBuffer(data)
		}()
	}
}
