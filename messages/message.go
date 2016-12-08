package messages

import (
	"errors"
	"net/http"
	//"time"
	//"strings"

	"github.com/julienschmidt/httprouter"

	"github.com/asiainfoLDP/datahub_commons/common"

	"github.com/asiainfoLDP/datafoundry_proxy/messages/notification"
)

//==================================================================
//
//==================================================================

func CreateInboxMessage(messageType, receiver, sender, hints string, level int, jsonData string) (int64, error) {
	db := getDB()
	if db == nil {
		return 0, errors.New("db not inited")
	}

	return notification.CreateMessage(db, messageType, receiver, sender, level, hints, jsonData)
}

func GetMessageByUserAndID(currentUserName string, messageid int64) (*notification.Message, error) {
	db := getDB()
	if db == nil {
		return nil, errors.New("db not inited")
	}

	return notification.GetMessageByUserAndID(db, currentUserName, messageid)
}

func ModifyMessageDataByID(messageid int64, jsonData string) error {
	db := getDB()
	if db == nil {
		return errors.New("db not inited")
	}

	return notification.ModifyMessageDataByID(db, messageid, jsonData)
}

//==================================================================
//
//==================================================================

func DeleteMessage(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	db := getDB()
	if db == nil {
		JsonResult(w, http.StatusInternalServerError, GetError(ErrorCodeDbNotInitlized), nil)
		return
	}

	currentUserName, e := MustCurrentUserName(r)
	if e != nil {
		JsonResult(w, http.StatusUnauthorized, e, nil)
		return
	}

	messageid, e := MustIntParamInPath(params, "id")
	if e != nil {
		JsonResult(w, http.StatusBadRequest, e, nil)
		return
	}

	/*
		m, err := common.ParseRequestJsonAsMap(r)
		if err != nil {
			JsonResult(w, http.StatusBadRequest, GetError2(ErrorCodeInvalidParameters, err.Error()), nil)
			return
		}

		//messageid, e := MustIntParamInMap (m, "messageid")
		//if e != nil {
		//	JsonResult(w, http.StatusBadRequest, e, nil)
		//	return
		//}

		action, e := MustStringParamInMap (m, "action", StringParamType_UrlWord)
		if e != nil {
			JsonResult(w, http.StatusBadRequest, e, nil)
			return
		}
	*/

	//r.ParseForm()

	err := notification.DeleteUserMessage(db, currentUserName, messageid)
	if err != nil {
		JsonResult(w, http.StatusInternalServerError, GetError2(ErrorCodeModifyMessage, err.Error()), nil)
		return
	}

	JsonResult(w, http.StatusOK, nil, nil)
}

func ModifyMessage(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	ModifyMessageWithCustomHandler(w, r, params, nil)
}

func defaultModifyMessageCustomHandler(r *http.Request, params httprouter.Params, m map[string]interface{}) (bool, *Error) {
	return false, nil
}

func ModifyMessageWithCustomHandler(w http.ResponseWriter, r *http.Request, params httprouter.Params,
	customF func(r *http.Request, params httprouter.Params, m map[string]interface{}) (bool, *Error)) {
	if customF == nil {
		customF = defaultModifyMessageCustomHandler
	}

	db := getDB()
	if db == nil {
		JsonResult(w, http.StatusInternalServerError, GetError(ErrorCodeDbNotInitlized), nil)
		return
	}

	currentUserName, e := MustCurrentUserName(r)
	if e != nil {
		JsonResult(w, http.StatusUnauthorized, e, nil)
		return
	}

	messageid, e := MustIntParamInPath(params, "id")
	if e != nil {
		JsonResult(w, http.StatusBadRequest, e, nil)
		return
	}

	m, err := common.ParseRequestJsonAsMap(r)
	if err != nil {
		JsonResult(w, http.StatusBadRequest, GetError2(ErrorCodeInvalidParameters, err.Error()), nil)
		return
	}

	//messageid, e := MustIntParamInMap (m, "messageid")
	//if e != nil {
	//	JsonResult(w, http.StatusBadRequest, e, nil)
	//	return
	//}

	action, e := MustStringParamInMap(m, "action", StringParamType_UrlWord)
	if e != nil {
		JsonResult(w, http.StatusBadRequest, e, nil)
		return
	}

	handled, err := notification.ModifyUserMessage(db, currentUserName, messageid, action)
	if handled {
		if err != nil {
			JsonResult(w, http.StatusBadRequest, GetError2(ErrorCodeModifyMessage, err.Error()), nil)
			return
		}
	} else if handled, e := customF(r, params, m); e != nil {
		JsonResult(w, http.StatusBadRequest, e, nil)
		return
	} else if !handled {
		JsonResult(w, http.StatusBadRequest, GetError2(ErrorCodeInvalidParameters, "not handled"), nil)
		return
	}

	JsonResult(w, http.StatusOK, nil, nil)
}

func GetMyMessages(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	db := getDB()
	if db == nil {
		JsonResult(w, http.StatusInternalServerError, GetError(ErrorCodeDbNotInitlized), nil)
		return
	}

	currentUserName, e := MustCurrentUserName(r)
	if e != nil {
		JsonResult(w, http.StatusUnauthorized, e, nil)
		return
	}

	r.ParseForm()

	level := notification.Level_Any
	if r.Form.Get("level") != "" {
		lvl, e := MustIntParamInQuery(r, "level")
		if e != nil {
			JsonResult(w, http.StatusBadRequest, e, nil)
			return
		}

		level = int(lvl)
	}

	status := notification.Status_Either
	if r.Form.Get("status") != "" {
		stts, e := MustIntParamInQuery(r, "status")
		if e != nil {
			JsonResult(w, http.StatusBadRequest, e, nil)
			return
		}
		status = int(stts)
		if status != notification.Status_Unread && status != notification.Status_Read {
			status = notification.Status_Either
		}
	}

	// message_type can be ""
	message_type := r.Form.Get("type")
	if message_type != "" {
		message_type, e = MustStringParamInQuery(r, "type", StringParamType_UrlWord)
		if e != nil {
			JsonResult(w, http.StatusBadRequest, e, nil)
			return
		}
	}

	// sender can be ""
	sender := r.Form.Get("sender")
	if sender != "" {
		// the sender may be email, or some special word, ex, @zhang3#aaa.com, $system, ....
		sender, e = MustStringParamInQuery(r, "sender", StringParamType_UnicodeUrlWord) //StringParamType_EmailOrUrlWord)
		if e != nil {
			JsonResult(w, http.StatusBadRequest, e, nil)
			return
		}
	}

	/*
		bt := r.Form.Get("beforetime")
		at := r.Form.Get("aftertime")
		if bt != "" && at != "" {
			JsonResult(w, http.StatusBadRequest, GetError2(ErrorCodeInvalidParameters, "beforetime and aftertime can't be both specified"), nil)
			return
		}

		var beforetime *time.Time = nil
		if bt != "" {
			// beforetime = &(optionalTimeParamInQuery(r, "beforetime", time.RFC3339, time.Now().Add(32*time.Hour)))
			// shit! above line doesn't work in golang
			t := optionalTimeParamInQuery(r, "beforetime", time.RFC3339, time.Now().Add(32*time.Hour))
			beforetime = &t
		}
		var aftertime *time.Time = nil
		if at != "" {
			t, _ := time.Parse("2006-01-02", "2000-01-01")
			t = optionalTimeParamInQuery(r, "aftertime", time.RFC3339, t)
			aftertime = &t
		}
	*/

	offset, size := optionalOffsetAndSize(r, 30, 1, 100)

	// /browser_messages, err := notification.GetUserMessagesForBrowser(db, currentUserName, message_type, status, sender, beforetime, aftertime)
	count, myMessages, err := notification.GetUserMessages(db, currentUserName, message_type, level, status, sender, offset, size)
	if err != nil {
		JsonResult(w, http.StatusInternalServerError, GetError2(ErrorCodeQueryMessage, err.Error()), nil)
		return
	}
	JsonResult(w, http.StatusOK, nil, newQueryListResult(count, myMessages))
}

func GetNotificationStats(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	db := getDB()
	if db == nil {
		JsonResult(w, http.StatusInternalServerError, GetError(ErrorCodeDbNotInitlized), nil)
		return
	}

	currentUserName, e := MustCurrentUserName(r)
	if e != nil {
		JsonResult(w, http.StatusUnauthorized, e, nil)
		return
	}

	r.ParseForm()

	// category can be ""
	category := r.Form.Get("category")
	if category != "" {
		category, e = MustStringParamInQuery(r, "category", StringParamType_UrlWord)
		if e != nil {
			JsonResult(w, http.StatusBadRequest, e, nil)
			return
		}
	}
	stat_category := notification.StatCategory_Unknown
	switch category {
	case "", "type":
		stat_category = notification.StatCategory_MessageType
	case "level":
		stat_category = notification.StatCategory_MessageLevel
	}
	if stat_category == notification.StatCategory_Unknown {
		JsonResult(w, http.StatusBadRequest, newInvalidParameterError("bad category param"), nil)
		return
	}

	message_stats, err := notification.RetrieveUserMessageStats(db, currentUserName, stat_category)
	if err != nil {
		JsonResult(w, http.StatusInternalServerError, GetError2(ErrorCodeGetMessageStats, err.Error()), nil)
		return
	}

	if len(message_stats) == 0 {
		message_stats = nil
	}

	JsonResult(w, http.StatusOK, nil, message_stats)
}

func ClearNotificationStats(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	db := getDB()
	if db == nil {
		JsonResult(w, http.StatusInternalServerError, GetError(ErrorCodeDbNotInitlized), nil)
		return
	}

	currentUserName, e := MustCurrentUserName(r)
	if e != nil {
		JsonResult(w, http.StatusUnauthorized, e, nil)
		return
	}

	_ = currentUserName
	JsonResult(w, http.StatusNotImplemented, GetError(ErrorCodeUrlNotSupported), nil)

	//err := notification.UpdateUserMessageStats(db, currentUserName, "", 0) // clear
	//if err != nil {
	//	JsonResult(w, http.StatusInternalServerError, GetError2(ErrorCodeResetMessageStats, err.Error()), nil)
	//	return
	//}

	//JsonResult(w, http.StatusOK, nil, nil)
}
