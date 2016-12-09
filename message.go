package main

import (
	//"database/sql"
	//_ "github.com/go-sql-driver/mysql"
	"encoding/json"
	"errors"
	"github.com/asiainfoLDP/datafoundry_proxy/messages"
	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"strings"
	//"github.com/asiainfoLDP/datafoundry_proxy/messages/notification"
	//"github.com/asiainfoLDP/datafoundry_serviceusage/usage"
	//"log"
	"io/ioutil"
	"os"
	"time"
	//"strconv"
	//"fmt"
	"fmt"
	"github.com/asiainfoLDP/datafoundry_payment/api/openshift"
	"github.com/asiainfoLDP/datahub_commons/common"
)

func GetMessages(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	glog.Infoln("from", r.RemoteAddr, r.Method, r.URL.RequestURI(), r.Proto)

	var username string
	var err error

	if username, err = authedIdentities(r); err != nil {
		RespError(w, err, http.StatusUnauthorized)
		return
	}

	r.Header.Set("User", username)

	messages.GetMyMessages(w, r, params)
}

func GetMessageStat(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	glog.Infoln("from", r.RemoteAddr, r.Method, r.URL.RequestURI(), r.Proto)

	var username string
	var err error

	if username, err = authedIdentities(r); err != nil {
		RespError(w, err, http.StatusUnauthorized)
		return
	}

	r.Header.Set("User", username)

	messages.GetNotificationStats(w, r, params)
}

func DeleteMessage(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	glog.Infoln("from", r.RemoteAddr, r.Method, r.URL.RequestURI(), r.Proto)

	var username string
	var err error

	if username, err = authedIdentities(r); err != nil {
		RespError(w, err, http.StatusUnauthorized)
		return
	}

	r.Header.Set("User", username)

	messages.DeleteMessage(w, r, params)
}

func ModifyMessage(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	glog.Infoln("from", r.RemoteAddr, r.Method, r.URL.RequestURI(), r.Proto)

	var username string
	var err error

	if username, err = authedIdentities(r); err != nil {
		RespError(w, err, http.StatusUnauthorized)
		return
	}

	r.Header.Set("User", username)

	messages.ModifyMessageWithCustomHandler(w, r, params, ModifyMessage_Custom)
}

func ModifyMessage_Custom(r *http.Request, params httprouter.Params, m map[string]interface{}) (bool, *messages.Error) {
	action, e := messages.MustStringParamInMap(m, "action", messages.StringParamType_UrlWord)
	if e != nil {
		return false, e
	}

	switch action {
	default:
		return false, nil
	case AcceptOrgInvitation:
		currentUserName, e := messages.MustCurrentUserName(r)
		if e != nil {
			return true, e
		}

		messageid, e := messages.MustIntParamInPath(params, "id")
		if e != nil {
			return true, e
		}

		msg, err := messages.GetMessageByUserAndID(currentUserName, messageid)
		if err != nil {
			return true, messages.GetError2(messages.ErrorCodeGetMessage, err.Error())
		}

		if strings.Index(msg.Hints, InviteMessage_Hints) < 0 {
			return true, messages.GetError2(messages.ErrorCodeInvalidParameters, "not an org invitation message")
		}

		im := &InviteMessage{}

		err = json.Unmarshal([]byte(msg.Raw_data), im)
		if err != nil {
			return true, messages.GetError2(messages.ErrorCodeInvalidParameters, err.Error())
		}

		if im.Accepted {
			return true, messages.GetError2(messages.ErrorCodeInvalidParameters, "already accepted")
		}

		im.Accepted = true

		jsondata, err := json.Marshal(im)
		if err != nil {
			return true, messages.GetError2(messages.ErrorCodeInvalidParameters, err.Error())
		}

		err = messages.ModifyMessageDataByID(messageid, string(jsondata))
		if err != nil {
			return true, messages.GetError2(messages.ErrorCodeInvalidParameters, err.Error())
		}
	}

	return true, nil
}

//===============================================================

const (
	MessageType_SiteNotify = "sitenotify" // may contains different messages, use message.hints to differentiate them
	//MessageType_AccountMsg = "accountmsg" // private message?
	//MessageType_Alert      = "alert"
	MessageType_OrderEvent = "orderevent"
)

const (
	Level_Any       = -1
	Level_General   = 0
	Level_Important = 50
)

//===============================================================

const InviteMessage_Hints = "invite,org"            // DON'T CHANGE!
const AcceptOrgInvitation = "accept_org_invitation" // DON'T CHANGE!
type InviteMessage struct {
	OrgID    string `json:"org_id"`
	OrgName  string `json:"org_name"`
	Accepted bool   `json:"accepted"`
}

func SendOrgInviteMessage(receiver, sender, orgId, orgName string) error {
	msg := &InviteMessage{
		OrgID:    orgId,
		OrgName:  orgName,
		Accepted: false,
	}

	jsonData, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	_, err = messages.CreateInboxMessage(
		MessageType_SiteNotify,
		receiver,
		sender,
		InviteMessage_Hints,
		Level_General,
		string(jsonData),
	)

	return err
}

//===============================================================

type Plan struct {
	Id              int64     `json:"id,omitempty"`
	Plan_id         string    `json:"plan_id,omitempty"`
	Plan_name       string    `json:"plan_name,omitempty"`
	Plan_type       string    `json:"plan_type,omitempty"`
	Plan_level      int       `json:"plan_level,omitempty"`
	Specification1  string    `json:"specification1,omitempty"`
	Specification2  string    `json:"specification2,omitempty"`
	Price           float32   `json:"price,omitempty"`
	Cycle           string    `json:"cycle,omitempty"`
	Region          string    `json:"region,omitempty"`
	Region_describe string    `json:"region_describe,omitempty"`
	Create_time     time.Time `json:"creation_time,omitempty"`
	Status          string    `json:"status,omitempty"`
}
type PurchaseOrder struct {
	Id                int64      `json:"id,omitempty"`
	Order_id          string     `json:"order_id,omitempty"`
	Account_id        string     `json:"namespace,omitempty"` // accountId
	Region            string     `json:"region,omitempty"`
	Plan_id           string     `json:"plan_id,omitempty"`
	Plan_type         string     `json:"_,omitempty"`
	Start_time        time.Time  `json:"start_time,omitempty"`
	End_time          time.Time  `json:"_,omitempty"`        // po
	EndTime           *time.Time `json:"end_time,omitempty"` // vo
	Deadline_time     time.Time  `json:"deadline,omitempty"`
	Last_consume_id   int        `json:"_,omitempty"`
	Ever_payed        int        `json:"_,omitempty"`
	Num_renew_retires int        `json:"_,omitempty"`
	Status            int        `json:"_,omitempty"`      // po
	StatusLabel       string     `json:"status,omitempty"` // vo
	Creator           string     `json:"creator,omitempty"`
	Resource_name     string     `json:"resource_name,omitempty"`
}
type MessageOrEmail struct {
	Reason string        `json:reason,omitempty`
	Order  PurchaseOrder `json:order,omitempty`
	Plan   *Plan         `json:plan,omitempty`
}

var AdminUser string

func init() {
	AdminUser = os.Getenv("MESSAGE_SENDER_ADMIN")
	if AdminUser == "" {
		glog.Fatal("MESSAGE_SENDER_ADMIN can't be blank")
	}
}
func CreateMassageOrEmail(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	glog.Infoln("from", r.RemoteAddr, r.Method, r.URL.RequestURI(), r.Proto)
	var username string
	var err error

	if username, err = authedIdentities(r); err != nil {
		RespError(w, err, http.StatusUnauthorized)
		glog.Info("don't have permission")
		return
	}
	if username != AdminUser {
		RespError(w, errors.New("permission denied"), http.StatusForbidden)
		glog.Info("is not an administrator")
		return
	}
	if r.Body == nil {
		glog.Fatal("no message")
		RespError(w, errors.New("no message"), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		glog.Error("readall is error")
		RespError(w, errors.New("readall is error"), http.StatusBadRequest)
		return
	}
	//r.ParseForm()
	//_type := r.Form.Get("type")
	_type := r.FormValue("type")
	switch _type {
	case MessageType_OrderEvent:
		var msg MessageOrEmail
		error := json.Unmarshal(data, &msg)
		if error != nil {
			RespError(w, errors.New("CreateMassageOrEmail Unmarshal error"), http.StatusBadRequest)
			glog.Fatal("CreateMassageOrEmail Unmarshal error")
			return
		}
		var level = Level_General
		switch msg.Reason {
		case "order_renew_failed", "order_closed":
			level = Level_Important
		}
		var admins = GetProjectAdmins(msg.Order.Account_id)
		for admin, adminValue := range admins {
			glog.Infof("k=%v ,v=%v", admin, adminValue)
			_, error = messages.CreateInboxMessage(MessageType_OrderEvent, adminValue, AdminUser, msg.Reason, level, string(data))

		}
		if error != nil {
			RespError(w, errors.New("CreateMassageOrEmail create message failed error"), http.StatusBadRequest)
			glog.Error("CreateMassageOrEmail create message failed error")
			return
		}
	default:
		RespError(w, errors.New("CreateMassageOrEmail  error"), http.StatusBadRequest)
		glog.Error("CreateMassageOrEmail  error")
		return
	}
	RespOK(w, nil)
	glog.Info("reseive success")

}

func GetProjectAdmins(projectName string) (adminstratorNmae []string) {
	url := fmt.Sprintf("%s/orgs/:project/roles", openshift.ListMembers)
	response, responsedata, err := common.RemoteCall("get", url, "", "")
	if err != nil {
		glog.Error("CreateMassageOrEmail create message failed error")
		return
	}
	if response.StatusCode != http.StatusOK {
		glog.Error("GetProjectAdmins error")
		return
	}
	glog.Info("GetProjectAdmins is success")

	return []string{string(responsedata)}
}
