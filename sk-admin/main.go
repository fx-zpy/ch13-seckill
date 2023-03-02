package main

import (
	"github.com/fx-zpy/SecKillGolang/pkg/bootstrap"
	conf "github.com/fx-zpy/SecKillGolang/pkg/config"
	"github.com/fx-zpy/SecKillGolang/pkg/mysql"
	"github.com/fx-zpy/SecKillGolang/sk-admin/setup"
)

func main() {
	mysql.InitMysql(conf.MysqlConfig.Host, conf.MysqlConfig.Port, conf.MysqlConfig.User, conf.MysqlConfig.Pwd, conf.MysqlConfig.Db) // conf.MysqlConfig.Db
	//setup.InitEtcd()
	setup.InitZk()
	setup.InitServer(bootstrap.HttpConfig.Host, bootstrap.HttpConfig.Port)

}
