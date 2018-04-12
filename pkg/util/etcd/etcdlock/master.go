// RAINBOND, Application Management Platform
// Copyright (C) 2014-2017 Goodrain Co., Ltd.

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version. For any non-GPL usage of Rainbond,
// one or multiple Commercial Licenses authorized by Goodrain Co., Ltd.
// must be obtained first.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

//Package etcdlock Master election using etcd.
package etcdlock

import (
	"context"
	"errors"
	"fmt"

	"github.com/Sirupsen/logrus"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/clientv3/concurrency"
)

//MasterEventType Various event types for the events channel.
type MasterEventType int

const (
	//MasterAdded this node has the lock.
	MasterAdded MasterEventType = iota
	//MasterDeleted MasterDeleted
	MasterDeleted
	//MasterModified MasterModified
	MasterModified
	//MasterError MasterError
	MasterError
)

// MasterEvent represents a single event sent on the events channel.
type MasterEvent struct {
	Type   MasterEventType // event type
	Master string          // identity of the lock holder
}

//MasterInterface Interface used by the etcd master lock clients.
type MasterInterface interface {
	// Start the election and attempt to acquire the lock. If acquired, the
	// lock is refreshed periodically based on the ttl.
	Start()

	// Stops watching the lock. Closes the events channel.
	Stop()

	// Returns the event channel used by the etcd lock.
	EventsChan() <-chan MasterEvent

	// Method to get the current lockholder. Returns "" if free.
	GetHolder() string
}

type masterLock struct {
	ctx           context.Context
	cancel        context.CancelFunc
	client        *clientv3.Client
	electionname  string
	prop          string
	etcdEndpoints []string
	election      *concurrency.Election
	session       *concurrency.Session
	eventchan     chan MasterEvent
	ttl           int64
	leaseID       clientv3.LeaseID
}

//CreateMasterLock  create master lock
func CreateMasterLock(etcdEndpoints []string, election string, prop string, ttl int64) (MasterInterface, error) {
	if etcdEndpoints == nil || len(etcdEndpoints) == 0 {
		etcdEndpoints = []string{"http://127.0.0.1:2379"}
	}
	ctx, cancel := context.WithCancel(context.Background())
	client, err := clientv3.New(clientv3.Config{
		Endpoints: etcdEndpoints,
		Context:   ctx,
	})
	if err != nil {
		cancel()
		return nil, fmt.Errorf("create etcd client error,%s", err.Error())
	}
	lease, err := client.Lease.Grant(ctx, ttl)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("create etcd lease error,%s", err.Error())
	}
	s, err := concurrency.NewSession(client, concurrency.WithContext(ctx), concurrency.WithLease(lease.ID))
	if err != nil {
		cancel()
		return nil, fmt.Errorf("new election session error,%s", err.Error())
	}
	e := concurrency.NewElection(s, election)
	ml := &masterLock{
		ctx:           ctx,
		cancel:        cancel,
		client:        client,
		electionname:  election,
		prop:          prop,
		etcdEndpoints: etcdEndpoints,
		election:      e,
		session:       s,
		eventchan:     make(chan MasterEvent, 2),
		leaseID:       lease.ID,
	}
	return ml, nil
}

// Campaign puts a value as eligible for the election. It blocks until
// it is elected, an error occurs, or the context is cancelled.
func (m *masterLock) campaign() error {
	logrus.Infof("start campaign master")
	if err := m.election.Campaign(m.ctx, m.prop); err != nil {
		return err
	}
	//elected
	logrus.Infof("current node is be elected master")
	select {
	case res := <-m.election.Observe(m.ctx):
		m.eventchan <- MasterEvent{Type: MasterAdded, Master: string(res.Kvs[0].Value)}
	case <-m.ctx.Done():
		return m.resign()
	case <-m.session.Done():
		m.eventchan <- MasterEvent{Type: MasterError, Master: ""}
		return errors.New("elect: session expired")
	}
	return nil
}
func (m *masterLock) resign() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	return m.election.Resign(ctx)
}
func (m *masterLock) Start() {
	go m.campaign()
}

func (m *masterLock) Stop() {
	m.cancel()
	m.resign()
}

func (m *masterLock) EventsChan() <-chan MasterEvent {
	return m.eventchan
}

func (m *masterLock) GetHolder() string {
	return ""
}