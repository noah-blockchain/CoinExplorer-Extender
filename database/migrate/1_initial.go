package migrate

import (
	"fmt"
	"github.com/go-pg/migrations"
)

const SqlCommand = ``

func init() {
	_ = migrations.Register(func(db migrations.DB) error {
		_, err := db.Exec(SqlCommand)
		if err != nil {
			return err
		}
		return nil
	}, func(db migrations.DB) error {
		fmt.Println("dropping scheme public...")
		_, err := db.Exec(`DROP SCHEME public CASCADE;`)
		return err
	})
}
