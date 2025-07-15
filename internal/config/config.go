package config

import (
	"context"
	"database/sql"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/Beguiler87/gator/internal/database"
	"github.com/google/uuid"
)

type Config struct {
	DbURL           string `json:"db_url"`
	CurrentUserName string `json:"current_user_name"`
}
type State struct {
	Config *Config
	DB     *database.Queries
	RawDB  *sql.DB
}
type Command struct {
	Name      string
	Arguments []string
}
type Commands struct {
	Commands map[string]func(*State, Command) error
}
type RSSFeed struct {
	Channel struct {
		Title       string    `xml:"title"`
		Link        string    `xml:"link"`
		Description string    `xml:"description"`
		Item        []RSSItem `xml:"item"`
	} `xml:"channel"`
}
type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

func getConfigFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	path := home + "/.gatorconfig.json"
	return path, nil
}

/*
Function not currently in use but do not wish to delete.

	func parseCommand(args []string) (Command, error) {
		if len(args) <= 1 {
			return Command{}, fmt.Errorf("error: no arguments provided")
		}
		name := args[1]
		arguments := args[2:]
		return Command{Name: name, Arguments: arguments}, nil
	}
*/
func (c *Commands) Register(name string, f func(*State, Command) error) {
	c.Commands[name] = f
}
func (c *Commands) Run(s *State, cmd Command) error {
	value, exists := c.Commands[cmd.Name]
	if exists {
		return value(s, cmd)
	}
	return fmt.Errorf("error: unknown command")
}
func (c *Commands) Users(s *State, cmd Command) error {
	userList, err := s.DB.GetUsers(context.Background())
	if err != nil {
		fmt.Println("Error: Failed to find users.")
		return nil
	}
	for _, user := range userList {
		if user.Name == s.Config.CurrentUserName {
			fmt.Printf("* %s (current)\n", user.Name)
		} else {
			fmt.Printf("* %s\n", user.Name)
		}
	}
	return nil
}
func (c *Commands) Agg(s *State, cmd Command) error {
	d, err := time.ParseDuration(cmd.Arguments[0])
	if err != nil {
		return err
	}
	ticker := time.NewTicker(d)
	defer ticker.Stop()
	fmt.Printf("Collecting feeds every %v\n", d)
	if err := ScrapeFeeds(context.Background(), s.RawDB); err != nil {
		fmt.Println("Error scraping feeds:", err)
	}
	for range ticker.C {
		if err := ScrapeFeeds(context.Background(), s.RawDB); err != nil {
			fmt.Println("Error scraping feeds:", err)
		}
	}
	return nil
}

func (c *Config) SetUser(username string) error {
	c.CurrentUserName = username
	cleanc, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	path, err := getConfigFilePath()
	if err != nil {
		return err
	}
	err = os.WriteFile(path, cleanc, 0o600)
	if err != nil {
		return err
	}
	return nil
}
func FetchFeed(ctx context.Context, feedURL string) (*RSSFeed, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", feedURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("user-Agent", "gator")
	client := http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	slice, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	feed := RSSFeed{}
	err = xml.Unmarshal(slice, &feed)
	if err != nil {
		return nil, err
	}
	feed.Channel.Title = html.UnescapeString(feed.Channel.Title)
	feed.Channel.Description = html.UnescapeString(feed.Channel.Description)
	for i := range feed.Channel.Item {
		feed.Channel.Item[i].Title = html.UnescapeString(feed.Channel.Item[i].Title)
		feed.Channel.Item[i].Description = html.UnescapeString(feed.Channel.Item[i].Description)
	}
	return &feed, nil
}

func Read() (Config, error) {
	path, err := getConfigFilePath()
	if err != nil {
		return Config{}, err
	}
	pathContents, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := json.Unmarshal(pathContents, &cfg); err != nil {
		return Config{}, err
	}
	cfg.DbURL = "postgres://postgres:postgres@localhost:5432/gator"
	return cfg, nil
}
func HandlerLogin(s *State, cmd Command) error {
	if len(cmd.Arguments) < 1 {
		return fmt.Errorf("username expected")
	}
	_, err := s.DB.GetUser(context.Background(), cmd.Arguments[0])
	if err != nil {
		fmt.Println("error: user does not exist")
		os.Exit(1)
	}
	if err := s.Config.SetUser(cmd.Arguments[0]); err != nil {
		return err
	}
	fmt.Printf("user set to %s\n", cmd.Arguments[0])
	return nil
}
func HandlerRegister(s *State, cmd Command) error {
	if len(cmd.Arguments) < 1 {
		return fmt.Errorf("error: username expected")
	}
	user, err := s.DB.CreateUser(context.Background(), database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      cmd.Arguments[0],
	})
	if err != nil {
		fmt.Println("error: user name already exists")
		os.Exit(1)
	}
	fmt.Printf("Created user: %+v", user)
	if err = s.Config.SetUser(cmd.Arguments[0]); err != nil {
		return err
	}
	fmt.Printf("User set to %s\n", cmd.Arguments[0])
	return nil
}
func HandlerReset(s *State, cmd Command) error {
	err := s.DB.DeleteUsers(context.Background())
	if err != nil {
		fmt.Println("Error: Failed to delete users.")
		os.Exit(1)
	}
	fmt.Println("Cleared users from database.")
	return nil
}
func HandlerAddFeed(s *State, cmd Command, user database.User) error {
	if len(cmd.Arguments) != 2 {
		return fmt.Errorf("username and url required")
	}
	name := cmd.Arguments[0]
	url := cmd.Arguments[1]
	newFeed, err := s.DB.CreateFeed(context.Background(), database.CreateFeedParams{
		Name:   sql.NullString{String: name, Valid: true},
		Url:    sql.NullString{String: url, Valid: true},
		UserID: uuid.NullUUID{UUID: user.ID, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("could not create feed: %w", err)
	}
	params := database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    uuid.NullUUID{UUID: user.ID, Valid: true},
		FeedID:    uuid.NullUUID{UUID: newFeed.ID, Valid: true},
	}
	_, err = s.DB.CreateFeedFollow(context.Background(), params)
	if err != nil {
		fmt.Printf("could not auto-follow feed: %v\n", err)
	}
	fmt.Printf("%v\n", newFeed)
	return nil
}
func HandlerFeeds(s *State, cmd Command) error {
	rows, err := s.DB.ListFeedsWithCreators(context.Background())
	if err != nil {
		return err
	}
	for _, row := range rows {
		fmt.Printf("%s %s %s\n", row.FeedName.String, row.Url.String, row.UserName.String)
	}
	return nil
}
func HandlerCreateFeedFollow(s *State, cmd Command, user database.User) error {
	if len(cmd.Arguments) < 1 {
		return fmt.Errorf("error: usage: follow <feed-url>")
	}
	ctx := context.Background()
	url := cmd.Arguments[0]
	feed, err := s.DB.GetFeedByURL(ctx, sql.NullString{String: url, Valid: true})
	if err != nil {
		return fmt.Errorf("error: Feed not found for URL: %s", url)
	}
	id := uuid.New()
	now := time.Now()
	params := database.CreateFeedFollowParams{
		ID:        id,
		CreatedAt: now,
		UpdatedAt: now,
		UserID:    uuid.NullUUID{UUID: user.ID, Valid: true},
		FeedID:    uuid.NullUUID{UUID: feed.ID, Valid: true},
	}
	result, err := s.DB.CreateFeedFollow(ctx, params)
	if err != nil {
		return fmt.Errorf("error: could not create follow: %w", err)
	}
	fmt.Printf("user %s is now following feed %s", result.UserName, result.FeedName.String)
	return nil
}
func HandlerFollowing(s *State, cmd Command, user database.User) error {
	ctx := context.Background()
	follows, err := s.DB.GetFeedFollowsForUser(ctx, uuid.NullUUID{UUID: user.ID, Valid: true})
	if err != nil {
		return fmt.Errorf("could not fetch follows: %w", err)
	}
	for _, follow := range follows {
		fmt.Println(follow.FeedName.String)
	}
	return nil
}
func MiddlewareLoggedIn(handler func(s *State, cmd Command, user database.User) error) func(*State, Command) error {
	return func(s *State, cmd Command) error {
		uName := s.Config.CurrentUserName
		if len(uName) < 1 {
			return fmt.Errorf("you must be logged in to use this command")
		}
		user, err := s.DB.GetUser(context.Background(), uName)
		if err != nil {
			return err
		}
		return handler(s, cmd, user)
	}
}
func HandlerUnfollow(s *State, cmd Command, user database.User) error {
	user_id := user.ID
	url := cmd.Arguments[0]
	feed_id, err := s.DB.GetFeed(context.Background(), sql.NullString{String: url, Valid: true})
	if err != nil {
		return err
	}
	params := database.UnfollowFeedParams{
		UserID: uuid.NullUUID{UUID: user_id, Valid: true},
		FeedID: uuid.NullUUID{UUID: feed_id, Valid: true},
	}
	err = s.DB.UnfollowFeed(context.Background(), params)
	if err != nil {
		return err
	}
	return nil
}
func stringToNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}
func ScrapeFeeds(ctx context.Context, db *sql.DB) error {
	queries := database.New(db)
	feed, err := queries.GetNextFeedToFetch(ctx)
	if err == sql.ErrNoRows {
		fmt.Println("DEBUG: No feeds to fetch")
		return nil
	} else if err != nil {
		return err
	}
	result, err := db.ExecContext(ctx,
		"UPDATE feed SET last_fetched_at = NOW(), updated_at = NOW() WHERE id = $1",
		feed.ID,
	)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		fmt.Println("no updates occurred, check that input is valid.")
		return nil
	}
	if !feed.Url.Valid {
		fmt.Println("feed URL is not valid, skipping fetch.")
		return nil
	}
	fmt.Printf("DEBUG: Found feed to scrape: %s\n", feed.Name.String)
	rss, err := FetchFeed(ctx, feed.Url.String)
	if err != nil {
		return err
	}
	fmt.Printf("DEBUG: Fetched RSS with %d items\n", len(rss.Channel.Item))
	for _, item := range rss.Channel.Item {
		publishedAt, err := time.Parse(time.RFC1123, item.PubDate)
		if err != nil {
			fmt.Printf("Error parsing date %s: %v\n", item.PubDate, err)
			continue
		}
		params := database.CreatePostParams{
			Title:       stringToNullString(item.Title),
			Url:         item.Link,
			Description: stringToNullString(item.Description),
			PublishedAt: publishedAt,
			FeedID:      feed.ID,
		}
		_, err = queries.CreatePost(ctx, params)
		if err != nil {
			fmt.Printf("Error creating post: %v\n", err)
			continue
		}
	}
	return nil
}
func HandlerBrowse(s *State, cmd Command, user database.User) error {
	limit := int32(2)
	if len(cmd.Arguments) > 0 {
		parsedLimit, err := strconv.Atoi(cmd.Arguments[0])
		if err != nil {
			return fmt.Errorf("invalid limit: %v", err)
		}
		limit = int32(parsedLimit)
	}
	queries := database.New(s.RawDB)
	posts, err := queries.GetPostsForUser(context.Background(), database.GetPostsForUserParams{
		UserID: uuid.NullUUID{UUID: user.ID, Valid: true},
		Limit:  limit,
	})
	if err != nil {
		return err
	}
	for _, post := range posts {
		title := "No Title"
		if post.Title.Valid {
			title = post.Title.String
		}
		fmt.Printf("Title: %s\n", title)
		fmt.Printf("URL: %s\n", post.Url)
		fmt.Printf("Published: %s\n", post.PublishedAt.Format("2006-01-02 15:04:05"))
		fmt.Println("---")
	}
	return nil
}
