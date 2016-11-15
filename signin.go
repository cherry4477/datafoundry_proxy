package main

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	oapi "github.com/openshift/origin/pkg/user/api/v1"
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
				RespError(w, ldpErrorNew(ErrCodeUnauthorized), http.StatusUnauthorized)
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
			defer TryCreatProject(auth, region.AccessToken, apiserver)
			return region, nil
		}
	} else {
		glog.Infoln("error from server, code:", stat)
	}
	return nil, ldpErrorNew(ErrCodeUnknownError)
}

func TryCreatProject(basic, bearerToken, apiserver string) {
	glog.Infoln("try create project with", basic, bearerToken, apiserver)
	var username string
	var apiHost string
	func() {

		b64auth := strings.Split(basic, " ")
		if len(b64auth) != 2 {
			glog.Infoln("basic string error.")
			return
		} else {
			payload, _ := base64.StdEncoding.DecodeString(b64auth[1])
			pair := strings.Split(string(payload), ":")
			if len(pair) != 2 {
				glog.Infoln(pair, "doesn't contain a username or password.")
				return
			} else {
				username = pair[0]
			}
		}

	}()

	func() {

		u, err := url.Parse(apiserver)
		if err != nil {
			glog.Error(err)
			return
		}

		fmt.Println("host:", u.Host)
		fmt.Println("path:", u.Path)
		fmt.Println("scheme", u.Scheme)

		apiHost = httpsAddrMaker(u.Host)

	}()

	func() {

		project_url := apiHost + "/oapi/v1/projectrequests"

		glog.Infoln(project_url)

		project := new(oapi.ProjectRequest)
		project.Name = username
		project.DisplayName = username
		project.Kind = "ProjectRequest"
		project.APIVersion = "v1"
		if reqbody, err := json.Marshal(project); err != nil {
			glog.Error(err)
		} else {
			tr := &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			}
			client := &http.Client{Transport: tr}
			glog.Infoln(string(reqbody))
			req, _ := http.NewRequest("POST", project_url, bytes.NewBuffer(reqbody))
			req.Header.Set("Authorization", "Bearer "+bearerToken)
			//log.Println(req.Header, bearer)

			resp, err := client.Do(req)
			if err != nil {
				glog.Error(err)
			} else {
				glog.Infoln(req.Host, req.Method, req.URL.RequestURI(), req.Proto, resp.StatusCode)
				b, _ := ioutil.ReadAll(resp.Body)
				glog.Infoln(string(b))
			}
		}

	}()
}
