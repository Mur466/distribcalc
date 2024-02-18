package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/Mur466/distribcalc/internal/cfg"
	l "github.com/Mur466/distribcalc/internal/logger"

)

var Conn *pgxpool.Pool

func InitDb() {

	var dbURL string = fmt.Sprintf("postgresql://%s:%s@%s:%d/%s", 
		cfg.Cfg.Dbuser,
		cfg.Cfg.Dbpassword,
		cfg.Cfg.Dbhost,
		cfg.Cfg.Dbport,
		cfg.Cfg.Dbname,
	)

	//fmt.Println(dbURL)
	
	var err error
	//Conn, err = pgx.Connect(context.Background(), dbURL)
	//db, err := pgxpool.New(ctx, connString)
	Conn, err = pgxpool.New(context.Background(), dbURL)
	if err != nil {
		l.Logger.Fatal(err.Error(),
			zap.String("dbURL",dbURL),
		)
	}

}

func ShutdownDb(){
	Conn.Close()
}
