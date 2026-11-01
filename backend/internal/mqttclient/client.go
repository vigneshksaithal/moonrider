package mqttclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sync/atomic"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"moonrider/backend/internal/domain"
	"moonrider/backend/internal/engine"
)

const Topic="moonrider/tractors/MR-001/telemetry"

func Env(key,fallback string) string {if v:=os.Getenv(key);v!=""{return v};return fallback}

func Options(clientID string) (*mqtt.ClientOptions,error) {
	broker:=Env("MQTT_BROKER","tcp://127.0.0.1:1883")
	o:=mqtt.NewClientOptions().AddBroker(broker).SetClientID(clientID).
		SetAutoReconnect(true).SetCleanSession(false).SetKeepAlive(20*time.Second).
		SetPingTimeout(5*time.Second).SetConnectTimeout(10*time.Second).
		SetMaxReconnectInterval(10*time.Second).SetOrderMatters(true)
	if len(broker)>=6 && broker[:6]=="ssl://" {
		cert,err:=tls.LoadX509KeyPair(os.Getenv("MQTT_CERT"),os.Getenv("MQTT_KEY"));if err!=nil{return nil,fmt.Errorf("client certificate: %w",err)}
		ca,err:=os.ReadFile(os.Getenv("MQTT_CA"));if err!=nil{return nil,fmt.Errorf("root CA: %w",err)}
		pool:=x509.NewCertPool();if !pool.AppendCertsFromPEM(ca){return nil,fmt.Errorf("invalid CA PEM")}
		o.SetTLSConfig(&tls.Config{MinVersion:tls.VersionTLS12,Certificates:[]tls.Certificate{cert},RootCAs:pool})
	}
	return o,nil
}

func Wait(ctx context.Context,t mqtt.Token) error {
	select {case <-ctx.Done():return ctx.Err();case <-t.Done():return t.Error()}
}

type Subscriber struct { client mqtt.Client; ready atomic.Bool }
func (s *Subscriber) Ready() bool{return s.ready.Load()}
func (s *Subscriber) Close(){s.ready.Store(false);s.client.Disconnect(500)}

func Subscribe(ctx context.Context,input chan<- engine.Delivery) (*Subscriber,error) {
	o,err:=Options(Env("MQTT_CLIENT_ID","moonrider-backend"));if err!=nil{return nil,err}
	s:=&Subscriber{}
	o.SetAutoAckDisabled(true)
	// The ordered callback intentionally blocks on a bounded queue. It performs
	// no client operations, so Paho's incoming callback cannot deadlock on an ACK.
	handler:=func(_ mqtt.Client,m mqtt.Message){
		if len(m.Payload())>4096 {slog.Warn("oversized MQTT event rejected");m.Ack();return}
		var r domain.Reading
		if err:=json.Unmarshal(m.Payload(),&r);err!=nil {slog.Warn("invalid MQTT JSON rejected");m.Ack();return}
		if err:=r.Validate(time.Now().UTC());err!=nil {slog.Warn("invalid telemetry rejected","error",err);m.Ack();return}
		select {case input<-engine.Delivery{Reading:r,Done:m.Ack}:case <-ctx.Done():}
	}
	o.SetDefaultPublishHandler(handler)
	o.SetConnectionLostHandler(func(_ mqtt.Client,err error){s.ready.Store(false);slog.Warn("MQTT connection lost","error",err)})
	o.SetOnConnectHandler(func(c mqtt.Client){
		s.ready.Store(false)
		// Paho invokes this callback in its own goroutine. Re-subscribe after every
		// connection, including if the broker expired the persistent session.
		work,cancel:=context.WithTimeout(ctx,10*time.Second);defer cancel()
		t:=c.Subscribe(Topic,1,handler)
		if err:=Wait(work,t);err!=nil {slog.Error("MQTT subscription failed","error",err);return}
		for _,q:=range t.(*mqtt.SubscribeToken).Result(){if q==0x80{slog.Error("MQTT subscription denied");return}}
		s.ready.Store(true);slog.Info("MQTT subscribed","topic",Topic)
	})
	s.client=mqtt.NewClient(o)
	work,cancel:=context.WithTimeout(ctx,12*time.Second);defer cancel()
	if err:=Wait(work,s.client.Connect());err!=nil{s.client.Disconnect(0);return nil,err}
	return s,nil
}
