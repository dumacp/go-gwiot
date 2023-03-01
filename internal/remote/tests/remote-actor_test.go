package tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/AsynkronIT/protoactor-go/actor"
	"github.com/dumacp/go-gwiot/internal/remote"
)

type MsgDisconnect struct{}
type MsgConnect struct{}

func TestNewRemote(t *testing.T) {

	remote.Dbpath = "/tmp/gwiotdb"

	disconnected := false

	props := actor.PropsFromFunc(func(ctx actor.Context) {

		switch ctx.Message().(type) {
		case *actor.Started:
			if ctx.Parent() != nil {
				ctx.Request(ctx.Parent(), &remote.MsgReconnect{})
			}
		case *remote.MsgSendData:
			time.Sleep(2800 * time.Millisecond)
			if ctx.Sender() != nil {
				if !disconnected {
					ctx.Respond(&remote.MsgAck{})
				} else {
					ctx.Respond(&remote.MsgError{
						Error: fmt.Errorf("Disconnected"),
					})
				}
			}
		case *MsgDisconnect:
			disconnected = true
		case *MsgConnect:
			disconnected = false
		}

	})

	type args struct {
		client *actor.Props
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "test1",
			args: args{
				client: props,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reclient := remote.NewRemote(tt.args.client)

			reclient.RetryDaysReplay(2)
			reclient.DisableReplay(false)

			propsRemote := actor.PropsFromFunc(reclient.Receive)

			rootctx := actor.NewActorSystem().Root

			pid, err := rootctx.SpawnNamed(propsRemote, tt.name)
			if err != nil {
				t.Errorf("create actor err: %s", err)
			}

			time.Sleep(15 * time.Second)
			rootctx.Send(pid, &MsgDisconnect{})
			time.Sleep(600 * time.Second)

		})
	}
}
