// goose connection string: goose postgres "postgres://postgres:postgres@localhost:5432/gator"
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
	cmds.Register("addfeed", config.HandlerAddFeed)
	cmds.Register("feeds", config.HandlerFeeds)
	cmds.Register("follow", config.HandlerCreateFeedFollow)
	cmds.Register("following", config.HandlerFollowing)
	cmdName := os.Args[1]
	cmdArgs := os.Args[2:]
	cmdStruct := config.Command{Name: cmdName, Arguments: cmdArgs}
	err = cmds.Run(s, cmdStruct)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
