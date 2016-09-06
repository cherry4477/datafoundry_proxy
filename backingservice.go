package main

import (
	"github.com/asiainfoLDP/datafoundry_proxy/openshift"
	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	oapi "github.com/openshift/origin/pkg/backingservice/api/v1"
	"net/http"
)

func BackingServiceHandler(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	bsName := params.ByName("backingservice")
	backingService := &oapi.BackingService{}
	ocClient := openshift.NewOpenshiftREST(nil)
	ocClient.OGet("/namespaces/openshift/backingservices/"+bsName, backingService)
	if ocClient.Err != nil {
		glog.Warningf("get backingservice %s error: %s", bsName, ocClient.Err)

		var err error
		var errCode int
		if ocClient.Status.Code == 0 {
			err = ocClient.Err
			errCode = http.StatusInternalServerError
		} else {
			err = ocClient
			errCode = ocClient.Status.Code
		}

		RespError(w, err, errCode)
		return
	}
	RespOK(w, backingService)
	return
}

func BackingServiceListHandler(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	bsList := &oapi.BackingServiceList{}
	ocClient := openshift.NewOpenshiftREST(nil)

	ocClient.OList("/namespaces/openshift/backingservices", nil, bsList)
	if ocClient.Err != nil {
		glog.Warningf("list backingservices error: %s", ocClient.Err)

		var err error
		var errCode int
		if ocClient.Status.Code == 0 {
			err = ocClient.Err
			errCode = http.StatusInternalServerError
		} else {
			err = ocClient
			errCode = ocClient.Status.Code
		}

		RespError(w, err, errCode)
		return
	}
	RespOK(w, bsList)
	return

}
