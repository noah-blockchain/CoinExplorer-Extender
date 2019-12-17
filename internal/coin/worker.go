package coin

import (
	"database/sql"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const (
	OperationTrxId int8 = iota
	OperationAddrId
)

type Item struct {
	ID            int
	Symbol        string
	MetaName      string
	OperationType int8
	CreatedAt     int64
}

func InitDB(filepath string) *sql.DB {
	db, err := sql.Open("sqlite3", filepath)
	if err != nil {
		panic(err)
	}
	if db == nil {
		panic("db nil")
	}
	return db
}

func CreateTable(db *sql.DB) error {
	// create table if not exists
	sqlTable := `
	CREATE TABLE IF NOT EXISTS items (
		id INTEGER NOT NULL PRIMARY KEY,
		symbol TEXT,
		meta_name TEXT,
		operation_type INTEGER,
		created_at INTEGER
	);
	`

	_, err := db.Exec(sqlTable)
	if err != nil {
		return err
	}
	return nil
}

func StoreItem(db *sql.DB, symbol, metaName string, operationType int8) error {
	tx, _ := db.Begin()
	stmt, _ := tx.Prepare("insert into items (symbol, meta_name, operation_type, created_at) values (?,?,?,?)")
	_, err := stmt.Exec(symbol, metaName, operationType, time.Now().Unix())
	if err != nil {
		return err
	}
	return tx.Commit()
}

func SelectItems(db *sql.DB) (*[]Item, error) {
	sqlSelect := `SELECT id, symbol, meta_name, operation_type FROM items`

	rows, err := db.Query(sqlSelect)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Item
	for rows.Next() {
		item := Item{}
		err := rows.Scan(&item.ID, &item.Symbol, &item.MetaName, &item.OperationType, &item.CreatedAt)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return &result, nil
}

func DeleteItem(db *sql.DB, id int) error {
	tx, _ := db.Begin()
	stmt, _ := tx.Prepare("delete from items where id=?")
	_, err := stmt.Exec(id)
	if err != nil {
		return err
	}
	return tx.Commit()
}
