package gwiot

import (
	"github.com/asynkron/protoactor-go/actor"
	renatsio "github.com/dumacp/go-gwiot/internal/remote-natsio"
)

func NewDiscoveryActor(topicReply string,
	funcSub func(topicReply string, pid *actor.PID, parse func(msg []byte) interface{}) error,
	funcPub func(topicDiscovery string, msg []byte)) actor.Actor {

	a := &renatsio.DiscoveryActor{
		FuncSub:    funcSub,
		FuncPub:    funcPub,
		TopicReply: topicReply,
	}

	return a
}
