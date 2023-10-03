package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"github.com/webmakom-com/saiStorage/config"
	"github.com/webmakom-com/saiStorage/mongo"
	"github.com/webmakom-com/saiStorage/websocket"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Server struct {
	Config      config.Configuration
	Websocket   bool
	Client      *mongo.Client
	DuplicateCh chan []byte
}

type AuthRequest struct {
	Collection string      `json:"collection"`
	Method     string      `json:"method"`
	Select     primitive.M `json:"select,omitempty"`
}

var ws websocket.Manager

func NewServer(c config.Configuration, w bool) Server {
	return Server{
		Config:      c,
		Websocket:   w,
		DuplicateCh: make(chan []byte),
	}
}

func (s Server) Start() {
	client, err := mongo.NewMongoClient(s.Config)

	if err != nil {
		fmt.Println("Could not connect to the mongo server:", err)
	}
	s.Client = &client
	defer s.Client.Host.Disconnect(context.Background())
	r := mux.NewRouter()
	http.Handle("/", r)

	if s.Websocket {
		r.HandleFunc("/ws/{any}", s.handleWSConnections)
		ws = websocket.NewWebSocketManager(s.Config)
	}

	if s.Config.Duplication {
		go s.duplicateRequests(context.TODO())
	}

	r.HandleFunc("/{any}", s.handleConnections)

	fmt.Println("Server has been started!")
	err = http.ListenAndServe(s.Config.HttpServer.Host+":"+s.Config.HttpServer.Port, nil)

	if err != nil {
		fmt.Println("Server error: ", err)
	}
}

func (s Server) StartHttps() {
	r := mux.NewRouter()
	http.Handle("/", r)

	if s.Websocket {
		r.HandleFunc("/ws", s.handleWSConnections)
		ws = websocket.NewWebSocketManager(s.Config)
	}

	r.HandleFunc("/{any}", s.handleConnections)

	fmt.Println("Server has been started!")

	httpsErr := http.ListenAndServeTLS(s.Config.HttpsServer.Host+":"+s.Config.HttpsServer.Port, "server.crt", "server.key", nil)

	if httpsErr != nil {
		fmt.Println("Server error: ", httpsErr)
	}
}

func (s Server) hasAccess(r *http.Request) bool {
	headers := r.Header
	token, ok := headers["Token"]

	if !ok {
		return false
	}

	if len(token) > 0 && token[0] == s.Config.Token {
		return true
	}

	return false
}

func (s Server) handleConnections(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	err := r.ParseForm()

	if err != nil {
		return
	}
	if s.Config.UsePermissionAuth {
		if !s.hasAccess(r) {
			w.WriteHeader(http.StatusUnauthorized)
			w.Header().Set("Content-Type", "application/json")
			fmt.Println("Unauthorized access")
			message, _ := json.Marshal(bson.M{"message": "Unauthorized access"})
			_, _ = w.Write(message)
			return
		}
	}

	s.handleServerRequest(w, r)
}

func (s Server) checkPermissionRequest(r *http.Request, collection, method string, selection primitive.M) error {
	headers := r.Header
	_, ok := headers["Token"]

	if !ok {
		return fmt.Errorf("empty token provided")
	}

	reqBody, err := json.Marshal(AuthRequest{
		Collection: collection,
		Method:     method,
		Select:     selection,
	})
	if err != nil {
		return err
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	accessURL := fmt.Sprintf("http://%s:%s/access", s.Config.SaiAuth.Host, s.Config.SaiAuth.Port)
	req, err := http.NewRequest("POST", accessURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return err
	}
	req.Header = headers

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if string(body) == "true" {
		return nil
	}
	return errors.New("Method or collection is not allowed\n")

}

// duplication request logic
func (s Server) duplicateRequests(ctx context.Context) {
	for {
		buf := <-s.DuplicateCh

		go func() {
			dupRequest := &duplicatedRequest{
				Data:   buf,
				Method: s.Config.DuplicateMethod,
			}

			dupRequestBytes, err := json.Marshal(dupRequest)
			if err != nil {
				return
			}

			req, err := http.NewRequest("POST", s.Config.DuplicationURL, bytes.NewBuffer(dupRequestBytes))
			if err != nil {
				return
			}

			client := http.Client{
				Timeout: time.Duration(s.Config.DuplicateTimeout) * time.Millisecond,
			}
			resp, err := client.Do(req)
			if err != nil {
				return
			}

			if resp.StatusCode != 200 {
				return
			}
		}()
	}
}
