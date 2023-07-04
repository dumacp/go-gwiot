package renatsio

import (
	"fmt"
	"testing"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/dumacp/go-gwiot/internal/utils"
	"github.com/dumacp/go-gwiot/pkg/gwiotmsg"
	"github.com/nats-io/nats.go"
)

func TestNatsActor_Receive(t *testing.T) {

	type fields struct {
		ctxroot  *actor.RootContext
		hostname string
		url      string
		jwtConf  *JwtConf
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
				ctxroot:  actor.NewActorSystem().Root,
				url:      nats.DefaultURL,
				hostname: "test1",
				// url:     "nats://fleet-nats.nebulae.com.co:443",
				jwtConf: nil,
				// jwtConf: &JwtConf{
				// 	User:         "dumawork",
				// 	Pass:         "dumawork",
				// 	Realm:        "DEVICES",
				// 	ClientID:     "devices3",
				// 	ClientSecret: "da9bbc28-01d8-43af-8c8a-fb0654937231",
				// 	KeycloakURL:  "https://fleet.nebulae.com.co/auth",
				// },
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
					&gwiotmsg.ListKeysBucket{
						Bucket: "AFC-PAYMENT-MEDIUM-TYPE",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			utils.SetHostname(tt.fields.hostname)
			a := NewClientNatsio(tt.fields.url, tt.fields.jwtConf, true)
			props := actor.PropsFromFunc(a.Receive)
			_, err := tt.fields.ctxroot.SpawnNamed(props, tt.name)
			if err != nil {
				t.Fatal(err)
			}
			time.Sleep(3 * time.Second)
			propsChild := actor.PropsFromFunc(NewChildNatsio(tt.name).Receive)
			pidChild, err := tt.fields.ctxroot.SpawnNamed(propsChild, fmt.Sprintf("%s-child", tt.name))
			if err != nil {
				t.Fatal(err)
			}

			for _, msg := range tt.args.messages {
				response, err := tt.fields.ctxroot.RequestFuture(pidChild, msg, 6*time.Second).Result()
				if err != nil {
					t.Errorf("error: %s", err)
					continue
				}
				t.Logf("response: %T, %s", response, response)
			}

			time.Sleep(10 * time.Second)

		})
	}
}
