package adapter

import (
	"database/sql"
	"fmt"
	"testing"

	go_ora "github.com/sijms/go-ora/v2"
)

func TestOracleAdapter_Connect(t *testing.T) {
	connStr := go_ora.BuildUrl("10.128.1.29", 1521, "nbsdb", "javatest", "nbs2o13", nil)
	fmt.Println(connStr)
	conn, err := sql.Open("oracle", connStr)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	err = conn.Ping()
	if err != nil {
		t.Fatal(fmt.Sprintf("Error pinging database: %v", err))
	}
	fmt.Println("Successfully connected to Oracle database!")
}
