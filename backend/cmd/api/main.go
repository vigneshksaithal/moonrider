package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"moonrider/backend/internal/domain"
	"moonrider/backend/internal/engine"
	"moonrider/backend/internal/httpapi"
	"moonrider/backend/internal/mqttclient"
	"moonrider/backend/internal/simulator"
	"moonrider/backend/internal/store"
)

func main(){if err:=run();err!=nil{slog.Error("startup failed","error",err);os.Exit(1)}}
func run() error {
	demo:=flag.Bool("demo",false,"in-memory cycling demo; bypasses MQTT and PostgreSQL")
	flag.Parse();slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout,nil)))
	ctx,stop:=signal.NotifyContext(context.Background(),os.Interrupt,syscall.SIGTERM);defer stop()
	var repo store.Repository
	if *demo{repo=store.NewMemory()}else{
		url:=os.Getenv("DATABASE_URL");if url==""{return startupError("DATABASE_URL is required (or use --demo)")}
		work,cancel:=context.WithTimeout(ctx,10*time.Second);defer cancel()
		p,err:=store.Open(work,url);if err!=nil{return err};repo=p
	}
	defer repo.Close()
	rules:=domain.DefaultRules()
	for name,target:=range map[string]*float64{"HOT_C":&rules.Hot,"COOL_C":&rules.Cool,"LOW_BATTERY_PCT":&rules.Low,"RECOVERED_BATTERY_PCT":&rules.Recovered}{
		if v:=os.Getenv(name);v!=""{n,err:=strconv.ParseFloat(v,64);if err!=nil{return err};*target=n}
	}
	for name,target:=range map[string]*time.Duration{"SUSTAIN_DURATION":&rules.Sustain,"OFFLINE_DURATION":&rules.Offline}{if v:=os.Getenv(name);v!=""{d,err:=time.ParseDuration(v);if err!=nil{return err};*target=d}}
	if rules.Cool>=rules.Hot||rules.Low>=rules.Recovered||rules.Sustain<=0||rules.Offline<=0{return startupError("invalid fault rules")}
	e,err:=engine.New(ctx,repo,rules,*demo);if err!=nil{return err}
	queue:=make(chan engine.Delivery,64)
	workerDone:=make(chan struct{});go func(){defer close(workerDone);e.Run(ctx,queue)}()
	defer func(){stop();<-workerDone}()
	ready:=func()bool{return true}
	if *demo{go demoLoop(ctx,queue)}else{sub,err:=mqttclient.Subscribe(ctx,queue);if err!=nil{return err};defer sub.Close();ready=sub.Ready}
	api:=&httpapi.Server{Engine:e,Repo:repo,MQTTReady:ready,Origins:strings.Split(mqttclient.Env("ALLOWED_ORIGINS","http://localhost:5173,http://127.0.0.1:5173"),",")}
	server:=&http.Server{Addr:mqttclient.Env("HTTP_ADDR","127.0.0.1:8080"),Handler:api.Handler(),ReadHeaderTimeout:5*time.Second,ReadTimeout:10*time.Second,IdleTimeout:60*time.Second,MaxHeaderBytes:16<<10}
	failed:=make(chan error,1);go func(){slog.Info("Moonrider listening","address",server.Addr,"demo",*demo);failed<-server.ListenAndServe()}()
	select{case <-ctx.Done():case err:=<-failed:if err!=http.ErrServerClosed{return err}}
	shutdown,cancel:=context.WithTimeout(context.Background(),5*time.Second);defer cancel()
	return server.Shutdown(shutdown)
}
type startupError string
func (e startupError) Error()string{return string(e)}
func demoLoop(ctx context.Context,q chan<-engine.Delivery){
	g:=simulator.New();ticker:=time.NewTicker(time.Second);defer ticker.Stop();var last *domain.Reading;second:=0
	for{select{case <-ctx.Done():return;case now:=<-ticker.C:
		scenario,send,replay:=simulator.Cycle(second);second++
		if !send{continue}
		if replay&&last!=nil{select{case q<-engine.Delivery{Reading:*last}:case <-ctx.Done():return}}
		r:=g.Next(scenario,now);last=&r;select{case q<-engine.Delivery{Reading:r}:case <-ctx.Done():return}
	}}
}
