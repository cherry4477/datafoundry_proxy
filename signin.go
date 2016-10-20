package main

import (
	"encoding/json"
	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	"net/http"
)

func SignIn(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	//fmt.Println("method:",r.Method)
	//fmt.Println("scheme", r.URL.Scheme)

	r.ParseForm()
	switch r.Method {
	case "GET":
		auth := r.Header.Get("Authorization")
		if len(auth) > 0 {
			glog.Infoln(auth)
			c := make(chan bool, 2)
			tokens := []RegionToken{}

			go func(c chan bool) {
				token, err := GetToken(auth, region_jd, true)
				if err == nil {
					token.Region = "cn-north-1"
					tokens = append(tokens, *token)
				}

				c <- true
			}(c)
			go func(c chan bool) {

				token, err := GetToken(auth, region_aws, false)
				if err == nil {
					token.Region = "cn-north-2"
					tokens = append(tokens, *token)
				}

				c <- true
			}(c)
			<-c
			<-c
			if len(tokens) > 0 {
				glog.Infoln(tokens)
				tokensBytes, _ := json.MarshalIndent(tokens, "", "  ")
				resphttp(w, http.StatusOK, tokensBytes)
			} else {
				RespError(w, ldpErrorNew(ErrCodeUnknownError), http.StatusInternalServerError)
			}

		} else {
			RespError(w, ldpErrorNew(ErrCodeUnauthorized), http.StatusUnauthorized)
		}
	case "OPTIONS":
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization")
		w.WriteHeader(http.StatusNoContent)
	default:
		RespError(w, ldpErrorNew(ErrCodeMethodNotAllowed), http.StatusMethodNotAllowed)
	}

}

type RegionToken struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   string `json:"expires_in"`
	TokenType   string `json:"token_type"`
	Region      string `json:"region"`
}

func GetToken(auth, apiserver string, extra bool) (*RegionToken, error) {
	token, stat := token_proxy(auth, apiserver, extra)
	if len(token) > 0 {
		region := new(RegionToken)
		if err := json.Unmarshal([]byte(token), region); err != nil {
			return nil, err
		} else {
			glog.Error(err)
			return region, nil
		}
	} else {
		glog.Infoln("error from server, code:", stat)
	}
	return nil, ldpErrorNew(ErrCodeUnknownError)
}
