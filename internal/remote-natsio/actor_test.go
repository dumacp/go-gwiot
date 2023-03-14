package renatsio

import (
	"testing"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/dumacp/go-gwiot/pkg/gwiotmsg"
	"github.com/nats-io/nats.go"
)

func TestNatsActor_Receive(t *testing.T) {

	type fields struct {
		ctxroot *actor.RootContext
		url     string
		jwtConf *JwtConf
	}
	type args struct {
		messages []interface{}
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
		{
			name: "test1",
			fields: fields{
				ctxroot: actor.NewActorSystem().Root,
				url:     nats.DefaultURL,
				jwtConf: nil,
				// replyFunc: func() {
				// 	exec.Command("nats", "")
				// },
			},
			args: args{
				messages: []interface{}{
					&gwiotmsg.Ping{},
					&gwiotmsg.RequestPubSub{
						Subject: "test.test1",
						Data:    []byte("hola mundo"),
					},
					&gwiotmsg.Request{
						Subject: "Replies.test2",
						Reply:   "Replies.test2reply",
						Data:    []byte("hola mundo"),
						Timeout: 5,
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewClientNatsio(tt.fields.url, tt.fields.jwtConf, true)
			props := actor.PropsFromFunc(a.Receive)
			pid, err := tt.fields.ctxroot.SpawnNamed(props, tt.name)
			if err != nil {
				t.Fatal(err)
			}

			for _, msg := range tt.args.messages {
				response, err := tt.fields.ctxroot.RequestFuture(pid, msg, 6*time.Second).Result()
				if err != nil {
					t.Errorf("error: %s", err)
					continue
				}
				t.Logf("response: %T, %s", response, response)
			}

		})
	}
}
