package main

import (
	"encoding/json"
	"fmt"
	"github.com/go-martini/martini"
	"github.com/gorilla/websocket"
	"github.com/martini-contrib/cors"
	"gopkg.in/mgo.v2/bson"
	"mongo"
	"net"
	"net/http"
	"sync"
)

func main() {
	m := martini.Classic()
	RouteInit(m)
	m.Run()
}

/******websocket相关函数******/

var ActiveClients = make(map[ClientConn]int)
var ActiveClientsRWMutex sync.RWMutex

type ClientConn struct {
	websocket *websocket.Conn
	clientIP  net.Addr
}

// 添加客户端
func addClient(cc ClientConn) {
	ActiveClientsRWMutex.Lock()
	ActiveClients[cc] = 0
	ActiveClientsRWMutex.Unlock()
}

//删除客户端
func deleteClient(cc ClientConn) {
	ActiveClientsRWMutex.Lock()
	delete(ActiveClients, cc)
	ActiveClientsRWMutex.Unlock()
}

//广播信息
func broadcastMessage(messageType int, message []byte) {
	ActiveClientsRWMutex.RLock()
	defer ActiveClientsRWMutex.RUnlock()

	for client, _ := range ActiveClients {
		if err := client.websocket.WriteMessage(messageType, message); err != nil {
			return
		}
	}
}

//接收消息
func readMsg(ws *websocket.Conn, sockCli ClientConn) {
	fmt.Println("[start readMsg]")
	// p := bson.M{}
	for {
		_, bp, err := ws.ReadMessage()
		if err != nil {
			deleteClient(sockCli)
			return
		}
		// if err := json.Unmarshal(bp, &p); err != nil {
		// 	return
		// }
		// p["state"] = 0

		// if err := mongo.D("msg").Add(p); err != nil {
		// 	return
		// }
		fmt.Println("[readMsg]", 1, string(bp))
		fmt.Println("[ActiveClients]", ActiveClients)
		broadcastMessage(1, bp)
	}
}

//将map转换为json字符串
func jsonEncode(m bson.M) string {
	b, _ := json.Marshal(m)
	return string(b)
}

//获取post或get的参数
func getParams(req *http.Request) bson.M {
	p := bson.M{}

	if req.Method == "GET" {
		params := req.URL.Query()
		for k, _ := range params {
			p[k] = params.Get(k)
		}
	} else if req.Method == "POST" {
		req.ParseForm()
		params := req.Form
		for k, _ := range params {
			p[k] = params.Get(k)
		}
	}

	return p
}

//路由规则初始化
func RouteInit(m *martini.ClassicMartini) {
	//CORS
	m.Use(cors.Allow(&cors.Options{
		AllowOrigins:     []string{"*"},
		AllowCredentials: true,
	}))

	m.Get("/Api/login", func(req *http.Request) string {
		p := getParams(req)

		user := *mongo.D("user").Find(p).One()
		if _, ok := user["_id"]; ok {
			user["uid"] = user["_id"]
			delete(user, "_id")
			result := bson.M{"code": 200, "msg": user}
			return jsonEncode(result)
		} else {
			result := bson.M{"code": 500, "msg": nil}
			return jsonEncode(result)
		}
	})

	// m.Get("/Api/logout", func(req *http.Request) string {
	// 	result := bson.M{"code": 200, "msg": "logout success"}
	// 	return jsonEncode(result)
	// })

	m.Get("/Api/user_chat_list", func(req *http.Request) string {
		p := getParams(req)

		user := *mongo.D("user").Find(p).One()
		if _, ok := user["_id"]; !ok {
			result := bson.M{"code": 404, "msg": "not login"}
			return jsonEncode(result)
		}

		uid := user["_id"].(bson.ObjectId)
		fmt.Println(uid.Hex())

		p = bson.M{"uid": bson.M{"$ne": uid.Hex()}}
		users := *mongo.D("chat_ing").Find(p).All()
		if len(users) > 0 {
			result := bson.M{"code": 200, "msg": users}
			return jsonEncode(result)
		} else {
			result := bson.M{"code": 500, "msg": "user list is empty"}
			return jsonEncode(result)
		}
	})

	m.Get("/socket", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("[socket start]")
		fmt.Println("[ActiveClients]", ActiveClients)
		ws, err := websocket.Upgrade(w, r, nil, 1024, 1024)
		if _, ok := err.(websocket.HandshakeError); ok {
			http.Error(w, "Not a websocket handshake", 400)
			return
		} else if err != nil {
			fmt.Println(err)
			return
		}
		client := ws.RemoteAddr()
		sockCli := ClientConn{ws, client}
		addClient(sockCli)
		go readMsg(ws, sockCli)

	})
}
