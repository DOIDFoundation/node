package store_test

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/DOIDFoundation/node/flags"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestCreateTable(t *testing.T) {
	homeDir := viper.GetString(flags.Home)
	dbFullPath := filepath.Join(homeDir, "data", "sqlite3.db")
	fmt.Println(dbFullPath)
	db, err := sql.Open("sqlite3", dbFullPath)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer db.Close()
}

func TestHelloName(t *testing.T) {
	// 打开 SQLite 数据库文件
	db, err := sql.Open("sqlite3", "./test.db")
	fmt.Println("----create db")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer db.Close()

	require.NotNil(t, db)

	// 创建表
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS users (id INTEGER PRIMARY KEY, name TEXT)")
	if err != nil {
		fmt.Println(err)
		return
	}

	// 插入数据
	result, err := db.Exec("INSERT INTO users (name) VALUES (?)", "Alice1")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(result.LastInsertId())

	// 查询数据
	rows, err := db.Query("SELECT id, name FROM users")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var name string
		err = rows.Scan(&id, &name)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println(id, name)
	}
}
