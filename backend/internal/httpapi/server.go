package httpapi

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"moonrider/backend/internal/domain"
	"moonrider/backend/internal/engine"
	"moonrider/backend/internal/store"
)

type Server struct { Engine *engine.Engine; Repo store.Repository; MQTTReady func()bool; Origins []string }
func (s *Server) Handler() http.Handler {
	mux:=http.NewServeMux()
	mux.HandleFunc("GET /healthz",func(w http.ResponseWriter,r *http.Request){respond(w,200,map[string]string{"status":"ok"})})
	mux.HandleFunc("GET /readyz",func(w http.ResponseWriter,r *http.Request){
		ctx,cancel:=context.WithTimeout(r.Context(),2*time.Second);defer cancel()
		if s.Repo.Ping(ctx)!=nil || !s.MQTTReady(){respond(w,503,map[string]string{"status":"not ready"});return};respond(w,200,map[string]string{"status":"ready"})
	})
	mux.HandleFunc("GET /api/tractors/MR-001",func(w http.ResponseWriter,r *http.Request){respond(w,200,s.Engine.Snapshot())})
	mux.HandleFunc("GET /api/alerts",func(w http.ResponseWriter,r *http.Request){respond(w,200,s.Engine.Snapshot().Alerts)})
	mux.HandleFunc("GET /api/live",s.live)
	return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
		start:=time.Now()
		defer func(){if v:=recover();v!=nil{slog.Error("HTTP panic","panic",v);http.Error(w,"internal error",500)};slog.Info("http request","method",r.Method,"path",r.URL.Path,"elapsed_ms",time.Since(start).Milliseconds())}()
		w.Header().Set("X-Content-Type-Options","nosniff")
		w.Header().Set("Cache-Control","no-store")
		if origin:=r.Header.Get("Origin");origin!="" {
			if !s.allowed(origin,r.Host){http.Error(w,"origin denied",403);return}
			w.Header().Set("Access-Control-Allow-Origin",origin);w.Header().Add("Vary","Origin")
			w.Header().Set("Access-Control-Allow-Methods","GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers","Content-Type")
		}
		if r.Method==http.MethodOptions{w.WriteHeader(204);return}
		mux.ServeHTTP(w,r)
	})
}
func (s *Server) allowed(origin,host string) bool {
	u,err:=url.Parse(origin);if err!=nil || (u.Scheme!="http"&&u.Scheme!="https"){return false}
	if u.Host==host{return true}
	for _,o:=range s.Origins{if strings.TrimSpace(o)==origin{return true}}
	return false
}
func respond(w http.ResponseWriter,status int,v any){w.Header().Set("Content-Type","application/json");w.WriteHeader(status);if err:=json.NewEncoder(w).Encode(v);err!=nil{slog.Warn("response write failed","error",err)}}
func (s *Server) live(w http.ResponseWriter,r *http.Request) {
	u:=websocket.Upgrader{CheckOrigin:func(r *http.Request)bool{return r.Header.Get("Origin")==""||s.allowed(r.Header.Get("Origin"),r.Host)}}
	c,err:=u.Upgrade(w,r,nil);if err!=nil{return};defer c.Close()
	ch,unsubscribe:=s.Engine.Subscribe();defer unsubscribe()
	done:=make(chan struct{})
	c.SetReadLimit(4096);_ = c.SetReadDeadline(time.Now().Add(45*time.Second))
	c.SetPongHandler(func(string)error{return c.SetReadDeadline(time.Now().Add(45*time.Second))})
	go func(){defer close(done);for{if _,_,err:=c.ReadMessage();err!=nil{return}}}()
	heartbeat:=time.NewTicker(5*time.Second);defer heartbeat.Stop()
	write:=func(snapshot domain.Snapshot)error{_ = c.SetWriteDeadline(time.Now().Add(5*time.Second));return c.WriteJSON(snapshot)}
	for {select{
	case <-done:return
	case <-r.Context().Done():return
	case snapshot:=<-ch:if write(snapshot)!=nil{return}
	case <-heartbeat.C:
		if err:=c.WriteControl(websocket.PingMessage,nil,time.Now().Add(3*time.Second));err!=nil{return}
		if write(s.Engine.Snapshot())!=nil{return}
	}}
}
