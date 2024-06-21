package renatsio

import (
	"fmt"
	"strings"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/dumacp/go-logs/pkg/logs"
)

func (a *ChildNats) addPrefix(ctx actor.Context, bucketName string) string {
	var bucket string
	if strings.HasPrefix(bucketName, "_") {
		if len(a.orgId) > 0 {
			bucket = fmt.Sprintf("%s%s", a.orgId, bucketName)
		} else if len(a.orgId) <= 0 && a.pidGwiot != nil {
			res, err := ctx.RequestFuture(a.pidGwiot, &GetOrgID{}, 1*time.Second).Result()
			if err != nil {
				logs.LogWarn.Printf("get orgId error: %s", err)
				if len(bucketName) > 2 {
					bucket = bucketName[1:]
				} else {
					bucket = bucketName
				}
			} else {
				switch res := res.(type) {
				case *GetOrgIDResponse:
					a.orgId = res.OrgID
					bucket = fmt.Sprintf("%s%s", a.orgId, bucketName)
				default:
					if len(bucketName) > 2 {
						bucket = bucketName[1:]
					} else {
						bucket = bucketName
					}
				}
			}
		} else {
			if len(bucketName) > 2 {
				bucket = bucketName[1:]
			} else {
				bucket = bucketName
			}
		}
	} else {
		bucket = bucketName
	}
	return bucket
}
