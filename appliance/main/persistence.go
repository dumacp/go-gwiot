package main

import (
	"log"

	"github.com/AsynkronIT/protoactor-go/persistence"
	pdb "github.com/dumacp/go-actors/persistence"
	"github.com/dumacp/go-gwiot/appliance/business/messages"
	"github.com/dumacp/go-gwiot/appliance/crosscutting/logs"
	"github.com/golang/protobuf/proto"
)

type provider struct {
	providerState persistence.ProviderState
}

var parseEvent = func(src []byte) proto.Message {
	i := new(messages.RemoteMSG)
	srcCopy := make([]byte, len(src))
	copy(srcCopy, src)
	err := proto.Unmarshal(srcCopy, i)
	if err != nil {
		log.Println(err)
		return nil
	}
	logs.LogBuild.Printf("recovery EVENT: %v", i)
	return i
}

var parseSnapshot = func(src []byte) proto.Message {
	i := new(messages.RemoteSnapshot)
	err := proto.Unmarshal(src, i)
	if err != nil {
		log.Println(err)
		return nil
	}
	logs.LogBuild.Printf("recovery SNAP: %v", i)
	return i
}

func newProvider(pathdb string, snapshotInterval int) (*provider, error) {
	db, err := pdb.NewBoltdbProvider(
		pathdb,
		snapshotInterval,
		parseEvent,
		parseSnapshot,
	)
	if err != nil {
		return nil, err
	}
	return &provider{
		providerState: db,
	}, nil
}

//GetState implementation for actor persistence
func (p *provider) GetState() persistence.ProviderState {
	return p.providerState
}
