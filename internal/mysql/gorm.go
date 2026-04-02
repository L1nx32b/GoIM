package mysql

import (
	"GoChatServer/internal/config"
	"GoChatServer/internal/model"
	"fmt"

	"GoChatServer/pkg/zaplog"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var GormDB *gorm.DB

func init() {
	// Mysql初始化
	// 获取全局配置
	conf := config.GetConfig().MysqlConfig
	user := conf.User
	password := conf.Password
	host := conf.Host
	port := conf.Port
	databaseName := conf.DatabaseName
	// 数据库名称(datasourcename)
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local", user, password, host, port, databaseName)

	var err error
	// 初始化GORM数据库连接
	GormDB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		zaplog.Fatal(err.Error())
	}

	// 自动迁移数据表
	err = GormDB.AutoMigrate(&model.UserInfo{}, &model.GroupInfo{}, &model.UserContact{}, &model.Session{}, &model.ContactApply{}, &model.Message{}) // 自动迁移，如果没有建表，会自动创建对应的表
	if err != nil {
		zaplog.Fatal(err.Error())
	}
}
