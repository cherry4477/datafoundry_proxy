package messages

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	//"github.com/julienschmidt/httprouter"

	_ "github.com/go-sql-driver/mysql"

	"github.com/asiainfoLDP/datahub_commons/log"

	"github.com/asiainfoLDP/datafoundry_proxy/env"

	"github.com/asiainfoLDP/datafoundry_proxy/messages/notification"
)

func init() {
	log_level := log.LevelString2Int(os.Getenv("LOG_LEVEL"))
	if log_level >= 0 {
		log.SetDefaultLoggerLevel(log_level)
	}
}

//======================================================
//
//======================================================

var Port int
var Debug = false
var Logger = log.DefaultlLogger()

var mysqlEnv, kafkaEnv, emailEnv env.Env

func Init( /*router *httprouter.Router, */ _mysqlEnv, _kafkaEnv, _emailEnv env.Env) bool {

	mysqlEnv, kafkaEnv, emailEnv = _mysqlEnv, _kafkaEnv, _emailEnv

	if initDB() == false {
		return false
	}

	//initRouter(router)
	//initGateWay()
	//initMQ()
	initMail()

	return true
}

/*
func initRouter(router *httprouter.Router) {
	//router.POST("/lapi/notifications", CreateMessage)
	router.GET("/lapi/notifications", GetMyMessages)
	//router.PUT("/notification/:messageid/:action", ModifyMessage)
	router.GET("/lapi/notification_stat", GetNotificationStats)
	//router.DELETE("/lapi/notification_stat", ClearNotificationStats)
}
*/

//=============================
//
//=============================

func MysqlAddrPort() (string, string) {
	//return os.Getenv(os.Getenv("ENV_NAME_MYSQL_ADDR")), os.Getenv(os.Getenv("ENV_NAME_MYSQL_PORT"))
	return mysqlEnv.Get("ENV_NAME_MYSQL_ADDR"), mysqlEnv.Get("ENV_NAME_MYSQL_PORT")
}

func MysqlDatabaseUsernamePassword() (string, string, string) {
	//return os.Getenv(os.Getenv("ENV_NAME_MYSQL_DATABASE")),
	//	os.Getenv(os.Getenv("ENV_NAME_MYSQL_USER")),
	//	os.Getenv(os.Getenv("ENV_NAME_MYSQL_PASSWORD"))
	return mysqlEnv.Get("ENV_NAME_MYSQL_DATABASE"),
		mysqlEnv.Get("ENV_NAME_MYSQL_USER"),
		mysqlEnv.Get("ENV_NAME_MYSQL_PASSWORD")
}

type Ds struct {
	db *sql.DB
}

var (
	ds = new(Ds)
)

func getDB() *sql.DB {
	if notification.IsServing() {
		return ds.db
	} else {
		return nil
	}
}

func initDB() bool {
	for i := 0; i < 3; i++ {
		connectDB()
		if ds.db == nil {
			select {
			case <-time.After(time.Second * 10):
				continue
			}
		} else {
			break
		}
	}

	if ds.db == nil {
		return false
	}

	upgradeDB()

	go updateDB()

	return true
}

func updateDB() {
	var err error
	ticker := time.Tick(5 * time.Second)
	for _ = range ticker {
		if ds.db == nil {
			connectDB()
		} else if err = ds.db.Ping(); err != nil {
			ds.db.Close()
			//ds.db = nil // draw snake feet
			connectDB()
		}
	}
}

func connectDB() {
	DB_ADDR, DB_PORT := MysqlAddrPort()
	DB_DATABASE, DB_USER, DB_PASSWORD := MysqlDatabaseUsernamePassword()

	DB_URL := fmt.Sprintf(`%s:%s@tcp(%s:%s)/%s?charset=utf8&parseTime=true`, DB_USER, DB_PASSWORD, DB_ADDR, DB_PORT, DB_DATABASE)

	Logger.Info("connect to ", DB_URL)
	db, err := sql.Open("mysql", DB_URL) // ! here, err is always nil, db is never nil.
	if err == nil {
		err = db.Ping()
	}

	if err != nil {
		Logger.Errorf("error: %s\n", err)
	} else {
		ds.db = db
	}
}

func upgradeDB() {
	//needUpgradeTables := os.Getenv("DONT_UPGRADE_MYSQL_TABLES") != "yes"
	needUpgradeTables := mysqlEnv.Get("DONT_UPGRADE_MYSQL_TABLES") != "yes"
	err := notification.TryToUpgradeDatabase(ds.db, "datafoundry:messages", needUpgradeTables) // don't change the name
	if err != nil {
		Logger.Errorf("TryToUpgradeDatabase error: %s", err.Error())
	}
}

/*
var (
	ApiGateway string

	RepositoryService   string
	SubscriptionSercice string
	UserService         string
	BillService         string
	DeamonService       string
)

func BuildServiceUrlPrefixFromEnv(name string, addrEnv string, portEnv string) string {
	addr := os.Getenv(addrEnv)
	if addr == "" {
		Logger.Errorf("%s env should not be null", addrEnv)
	}
	port := os.Getenv(portEnv)

	prefix := ""
	if port == "" {
		prefix = fmt.Sprintf("http://%s", addr)
	} else {
		prefix = fmt.Sprintf("http://%s:%s", addr, port)
	}

	Logger.Infof("%s = %s", name, prefix)

	return prefix
}


func initGateWay() {
	RepositoryService = BuildServiceUrlPrefixFromEnv("RepositoryService", "REPOSIROTY_SERVICE_API_SERVER", "REPOSIROTY_SERVICE_API_PORT")
	SubscriptionSercice = BuildServiceUrlPrefixFromEnv("SubscriptionSercice", "SUBSCRIPTION_SERVICE_API_SERVER", "SUBSCRIPTION_SERVICE_API_PORT")
	UserService = BuildServiceUrlPrefixFromEnv("UserService", "USER_SERVICE_API_SERVER", "USER_SERVICE_API_PORT")
	BillService = BuildServiceUrlPrefixFromEnv("BillService", "BILL_SERVICE_API_SERVER", "BILL_SERVICE_API_PORT")
	DeamonService = BuildServiceUrlPrefixFromEnv("DeamonService", "DEAMON_SERVICE_API_SERVER", "DEAMON_SERVICE_API_PORT")
}

func getRepositoryService() string {
	return RepositoryService
}

func getSubscriptionSercice() string {
	return SubscriptionSercice
}

func getUserService() string {
	return UserService
}

func getBillService() string {
	return BillService
}

func getDeamonService() string {
	return DeamonService
}
*/
