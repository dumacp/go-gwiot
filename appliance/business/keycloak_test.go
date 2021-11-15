package business

import (
	"context"
	"reflect"
	"testing"

	"github.com/dumacp/keycloak"
	"golang.org/x/oauth2"
)

func Test_tokenSorce(t *testing.T) {
	type args struct {
		ctx      context.Context
		keyc     keycloak.Keycloak
		username string
		password string
	}
	tests := []struct {
		name    string
		args    args
		want    oauth2.TokenSource
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "test1",
			args: args{
				context.Background(),
				nil,
				Hostname(),
				Hostname(),
			},
			want:    nil,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			config := newKeyConfig()
			httpContext := newHTTPContext(context.Background(), nil)
			keyc, _ := keycServer(httpContext, config)
			tt.args.keyc = keyc

			got, err := tokenSorce(tt.args.ctx, tt.args.keyc, tt.args.username, tt.args.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("tokenSorce() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				tk, _ := got.Token()
				t.Errorf("tokenSorce() = %#v, %#v, %v, want %v", got, tk, tk.Expiry, tt.want)
			}
		})
	}
}
