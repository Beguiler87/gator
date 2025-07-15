// goose migration/connection string. Run: goose -dir sql/schema postgres "postgres://postgres:postgres@localhost:5432/gator" and add up or down depending.
package main

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/Beguiler87/gator/internal/config"
	"github.com/Beguiler87/gator/internal/database"
	_ "github.com/lib/pq"
)

func main() {
	incoming, err := config.Read()
	if err != nil {
		fmt.Println("error reading config:", err)
		return
	}
	db, err := sql.Open("postgres", incoming.DbURL)
	if err != nil {
		fmt.Println("Error: Invalid postgres URL.")
		return
	}
	dbQueries := database.New(db)
	s := &config.State{
		Config: &incoming,
		DB:     dbQueries,
		RawDB:  db,
	}
	cmds := &config.Commands{
		Commands: make(map[string]func(*config.State, config.Command) error),
	}
	cmds.Register("login", config.HandlerLogin)
	if len(os.Args) < 2 {
		fmt.Println("Error: Please provide one or more arguments.")
		os.Exit(1)
	}
	cmds.Register("register", config.HandlerRegister)
	cmds.Register("reset", config.HandlerReset)
	cmds.Register("users", cmds.Users)
	cmds.Register("agg", cmds.Agg)
	cmds.Register("feeds", config.HandlerFeeds)
	cmds.Register("addfeed", config.MiddlewareLoggedIn(config.HandlerAddFeed))
	cmds.Register("follow", config.MiddlewareLoggedIn(config.HandlerCreateFeedFollow))
	cmds.Register("following", config.MiddlewareLoggedIn(config.HandlerFollowing))
	cmds.Register("unfollow", config.MiddlewareLoggedIn(config.HandlerUnfollow))
	cmds.Register("browse", config.MiddlewareLoggedIn(config.HandlerBrowse))
	cmdName := os.Args[1]
	cmdArgs := os.Args[2:]
	cmdStruct := config.Command{Name: cmdName, Arguments: cmdArgs}
	err = cmds.Run(s, cmdStruct)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
