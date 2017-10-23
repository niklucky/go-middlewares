package middlewares

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"

	"github.com/julienschmidt/httprouter"
)

/*
HTTPServer - duh!
*/
type HTTPServer struct {
	Host   string
	Port   int
	router *httprouter.Router
}

type httpHandler func(httprouter.Params, []byte) (interface{}, error)

/*
NewHTTPServer - constructor of duh!
*/
func NewHTTPServer(h string, p int) HTTPServer {
	return HTTPServer{
		Host:   h,
		Port:   p,
		router: httprouter.New(),
	}
}

/*
AddHandler - adding handlers for handling (duuh!)
*/
func (srv *HTTPServer) AddHandler(method string, route string, h httpHandler) {
	if method == "GET" {
		srv.router.GET(route, srv.baseHandler(h))
	}
	if method == "POST" {
		srv.router.POST(route, srv.baseHandler(h))
	}
	if method == "PUT" {
		srv.router.PUT(route, srv.baseHandler(h))
	}
	if method == "DELETE" {
		srv.router.DELETE(route, srv.baseHandler(h))
	}
}

/*
Start - stopping server (duuh!)
*/
func (srv *HTTPServer) Start() {
	fmt.Println("Starting server: ", srv.getHost())
	log.Fatal(http.ListenAndServe(srv.getHost(), srv.router))
}

func (srv *HTTPServer) getHost() string {
	return srv.Host + ":" + strconv.Itoa(srv.Port)
}

func (srv *HTTPServer) SendError(w http.ResponseWriter, httpCode int, err error) {
	e := make(map[string]interface{})
	e["error"] = err
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpCode)
	b, _ := json.Marshal(e)
	w.Write(b)
}

func (srv *HTTPServer) baseHandler(h httpHandler) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			srv.SendError(w, 500, err)
			return
		}
		result, err := h(ps, body)
		if err != nil {
			srv.SendError(w, 400, err)
			return
		}
		res := make(map[string]interface{})
		res["data"] = result

		resp, err := json.Marshal(res)
		if err != nil {
			srv.SendError(w, 400, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(resp)
	}
}
