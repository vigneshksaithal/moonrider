package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"moonrider/backend/internal/domain"
	"moonrider/backend/internal/mqttclient"
	"moonrider/backend/internal/simulator"
)
func main(){if err:=run();err!=nil{slog.Error("simulator stopped","error",err);os.Exit(1)}}
func run()error {
	scenario:=flag.String("scenario","cycle","normal, overheat, low-battery, or cycle")
	duplicate:=flag.Bool("duplicate",false,"send each reading twice with the same event ID")
	flag.Parse()
	if *scenario!="normal"&&*scenario!="overheat"&&*scenario!="low-battery"&&*scenario!="cycle"{return fmt.Errorf("unknown scenario %q",*scenario)}
	ctx,stop:=signal.NotifyContext(context.Background(),os.Interrupt,syscall.SIGTERM);defer stop()
	o,err:=mqttclient.Options(mqttclient.Env("MQTT_CLIENT_ID","moonrider-simulator"));if err!=nil{return err}
	c:=mqtt.NewClient(o);defer c.Disconnect(500)
	connect,cancel:=context.WithTimeout(ctx,12*time.Second);defer cancel()
	if err:=mqttclient.Wait(connect,c.Connect());err!=nil{return err}
	publish:=func(r domain.Reading)error{b,err:=json.Marshal(r);if err!=nil{return err};work,cancel:=context.WithTimeout(ctx,10*time.Second);defer cancel();return mqttclient.Wait(work,c.Publish(mqttclient.Topic,1,false,b))}
	ticker:=time.NewTicker(time.Second);defer ticker.Stop();g:=simulator.New();var last *domain.Reading;second:=0
	for{select{case <-ctx.Done():return nil;case now:=<-ticker.C:
		send,replay:=true,false;mode:=*scenario
		if mode=="cycle"{mode,send,replay=simulator.Cycle(second)};second++
		if !send{slog.Info("simulating silent tractor uplink");continue}
		if replay&&last!=nil{if err:=publish(*last);err!=nil{slog.Warn("replay failed","error",err)}}
		r:=g.Next(mode,now);last=&r
		if err:=publish(r);err!=nil{slog.Warn("publish failed","error",err);continue}
		if *duplicate{if err:=publish(r);err!=nil{slog.Warn("duplicate publish failed","error",err)}}
		slog.Info("telemetry sent","scenario",mode,"event_id",r.EventID,"temperature",r.Temperature,"battery",r.Battery)
	}}
}
