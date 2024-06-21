package renatsio

import (
	"fmt"
	"strings"
)

func (a *NatsActor) getOrgID() string {
	if len(a.orgID) > 0 {
		return a.orgID
	} else if a.userInfo == nil {
		return ""
	} else {
		// fmt.Printf("claims: %v\n", a.userInfo.Claims)
		claims := make(map[string]interface{})
		if err := a.userInfo.Claims(&claims); err != nil {
			return ""
		} else {
			fmt.Printf("claims: %v\n", claims)
			if v, ok := claims["group_name"]; ok {
				if v, ok := v.(string); ok {
					sp := strings.Split(v, "_")
					if len(sp) < 2 {
						return ""
					} else {
						a.orgID = sp[1]
						return sp[1]
					}
				}
			} else {
				return ""
			}
		}
	}
	return ""
}

func (a *NatsActor) getGorupName() string {
	if len(a.groupName) > 0 {
		return a.groupName
	} else if a.userInfo == nil {
		return ""
	} else {
		// fmt.Printf("claims: %v\n", a.userInfo.Claims)
		claims := make(map[string]interface{})
		if err := a.userInfo.Claims(&claims); err != nil {
			return ""
		} else {
			fmt.Printf("claims: %v\n", claims)
			if v, ok := claims["group_name"]; ok {
				if v, ok := v.(string); ok {
					a.groupName = v
					return v
				}
			} else {
				return ""
			}
		}
	}
	return ""
}
