package engine

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"moonrider/backend/internal/domain"
	"moonrider/backend/internal/store"
)

type Engine struct {
	mu sync.Mutex
	repo store.Repository
	rules domain.Rules
	state domain.Snapshot
	subs map[chan domain.Snapshot]bool
}

func New(ctx context.Context,repo store.Repository,rules domain.Rules,demo bool) (*Engine,error) {
	s,err:=repo.Load(ctx);if err!=nil{return nil,err};s.Demo=demo
	return &Engine{repo:repo,rules:rules,state:s,subs:map[chan domain.Snapshot]bool{}},nil
}
func (e *Engine) Snapshot() domain.Snapshot { e.mu.Lock();defer e.mu.Unlock();return domain.Clone(e.state) }
func (e *Engine) Subscribe() (<-chan domain.Snapshot,func()) {
	e.mu.Lock();defer e.mu.Unlock();ch:=make(chan domain.Snapshot,1);e.subs[ch]=true;ch<-domain.Clone(e.state)
	return ch,func(){e.mu.Lock();defer e.mu.Unlock();delete(e.subs,ch)}
}
func (e *Engine) commit(ctx context.Context,r *domain.Reading,s domain.Snapshot) error {
	s.Version++
	if len(s.Alerts)>100{s.Alerts=s.Alerts[len(s.Alerts)-100:]}
	ok,err:=e.repo.Save(ctx,r,s);if err!=nil{return err};if !ok{return nil}
	e.state=s
	for ch:=range e.subs {
		// Slow clients get the most recent complete snapshot, not an unbounded backlog.
		select {case <-ch:default:}
		ch<-domain.Clone(s)
	}
	return nil
}
func (e *Engine) Process(ctx context.Context,r domain.Reading,now time.Time) error {
	if err:=r.Validate(now);err!=nil{return err}
	e.mu.Lock();defer e.mu.Unlock()
	// A replay is not a heartbeat. Old data cannot restore connectivity or rewind state.
	if now.Sub(r.Timestamp)>e.rules.Offline{return nil}
	if e.state.Reading!=nil && !r.Timestamp.After(e.state.Reading.Timestamp){return nil}
	s:=domain.Clone(e.state)
	if s.LastSeen==nil || now.Sub(*s.LastSeen)>=e.rules.Offline {s.HotSince=nil;s.CoolSince=nil}
	s.Reading=&r;s.Online=true;s.LastSeen=&now
	setFault(&s,"offline",false,"Lost connectivity","No fresh tractor telemetry received.",now)
	if r.Temperature>e.rules.Hot {
		s.CoolSince=nil
		if s.HotSince==nil{s.HotSince=&now}
		if now.Sub(*s.HotSince)>=e.rules.Sustain{setFault(&s,"overheating",true,"Motor overheating",fmt.Sprintf("Motor above %.0f°C for %s.",e.rules.Hot,e.rules.Sustain),now)}
	} else { s.HotSince=nil }
	if r.Temperature<e.rules.Cool {
		if s.CoolSince==nil{s.CoolSince=&now}
		if now.Sub(*s.CoolSince)>=e.rules.Sustain{setFault(&s,"overheating",false,"","",now)}
	} else {s.CoolSince=nil}
	if r.Battery<e.rules.Low {setFault(&s,"low_battery",true,"Low battery",fmt.Sprintf("Battery below %.0f%%. Plan a charging stop.",e.rules.Low),now)}
	if r.Battery>e.rules.Recovered{setFault(&s,"low_battery",false,"","",now)}
	return e.commit(ctx,&r,s)
}
func (e *Engine) CheckOffline(ctx context.Context,now time.Time) error {
	e.mu.Lock();defer e.mu.Unlock()
	if e.state.LastSeen==nil || !e.state.Online || now.Sub(*e.state.LastSeen)<e.rules.Offline{return nil}
	s:=domain.Clone(e.state);s.Online=false;s.HotSince=nil;s.CoolSince=nil
	setFault(&s,"offline",true,"Lost connectivity","No fresh tractor telemetry received.",now)
	return e.commit(ctx,nil,s)
}
func setFault(s *domain.Snapshot,kind string,active bool,title,message string,now time.Time) {
	for i:=range s.Alerts {
		if s.Alerts[i].Kind==kind && s.Alerts[i].ResolvedAt==nil {
			if !active{s.Alerts[i].ResolvedAt=&now};return
		}
	}
	if active {s.Alerts=append(s.Alerts,domain.Alert{ID:fmt.Sprintf("%s-%d",kind,now.UnixNano()),Kind:kind,Title:title,Message:message,OpenedAt:now})}
}

type Delivery struct { Reading domain.Reading; Done func() }

// One worker preserves per-tractor ordering. Parallelize by tractor partition when
// adding a fleet, rather than allowing workers to race on one tractor's state.
func (e *Engine) Run(ctx context.Context,input <-chan Delivery) {
	ticker:=time.NewTicker(time.Second);defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():return
		case d,ok:=<-input:
			if !ok{return}
			for {
				work,cancel:=context.WithTimeout(ctx,5*time.Second)
				err:=e.Process(work,d.Reading,time.Now().UTC());cancel()
				if err==nil{if d.Done!=nil{d.Done()};break}
				slog.Error("telemetry processing failed; retrying without ACK", "error",err)
				select {case <-ctx.Done():return;case <-time.After(time.Second):}
			}
		case now:=<-ticker.C:
			work,cancel:=context.WithTimeout(ctx,5*time.Second)
			if err:=e.CheckOffline(work,now.UTC());err!=nil{slog.Error("offline check failed","error",err)};cancel()
		}
	}
}
