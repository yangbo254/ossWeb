package ossweb

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/golang-jwt/jwt/v5"
)

type Auth struct {
	Ver       int    `json:"ver"`
	Username  string `json:"username"`
	Userid    int    `json:"userid"`
	AuthType  string `json:"type"`
	Usertoken string `json:"token"`
}

var hmacSecret string = "5dd89def9f73d4b6d7e8ae874e61cbaa990dc6b436e4edcb5496e5d997ca8fc4"
var tokenCheckServer string = "http://userauth/api/backend/checktoken"

func (pAuth *Auth) CheckToken(Authorization string) error {

	token, err := jwt.Parse(Authorization, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(hmacSecret), nil
	})

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		ver, ok := claims["ver"].(int)
		if !ok {
			return err
		}
		pAuth.Ver = ver
		if ver > 1 {
			userName, ok := claims["username"].(string)
			if !ok {
				return err
			}
			userId, ok := claims["userid"].(int)
			if !ok {
				return err
			}
			authType, ok := claims["type"].(string)
			if !ok {
				return err
			}
			authToken, ok := claims["token"].(string)
			if !ok {
				return err
			}
			pAuth.Username = userName
			pAuth.Userid = userId
			pAuth.AuthType = authType
			pAuth.Usertoken = authToken
		}

		if pAuth.AuthType == "userauth" {
			type tokenCheckReq struct {
				Username string `json:"username"`
				Userid   int    `json:"userid"`
				Token    string `json:"token"`
			}
			type tokenCheckAck struct {
				Code int    `json:"code"`
				Msg  string `json:"msg"`
			}
			dataAck := &tokenCheckAck{}
			dataReq := &tokenCheckReq{
				Username: pAuth.Username,
				Userid:   pAuth.Userid,
				Token:    pAuth.Usertoken,
			}
			client := &http.Client{}
			bytesData, _ := json.Marshal(dataReq)
			req, _ := http.NewRequest("POST", tokenCheckServer, bytes.NewReader(bytesData))
			resp, _ := client.Do(req)
			body, _ := io.ReadAll(resp.Body)
			if err := json.Unmarshal(body, dataAck); err != nil {
				return err
			}
			if dataAck.Code != 0 {
				return errors.New(dataAck.Msg)
			}
			return nil
		}

	} else {
		fmt.Println(err)
		return err
	}
	return errors.New("unknown")
}
