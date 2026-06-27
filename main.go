
package main

import (
	"context"
	"database/sql"
	"encoding/xml"
	"net/http"
	"strings"
	"fmt"
	"os"
	"io"
	"time"
	"html"

	"github.com/google/uuid"
	
	"github.com/CKojiro/go_aggregator/internal/config"
	"github.com/CKojiro/go_aggregator/internal/database"
	
	_ "github.com/lib/pq"
)

type state struct {
	db *database.Queries
	cfg *config.Config
}

type command struct {
	name string
	args []string
}

type commands struct {
	handlers map[string]func(*state, command) error
}

type RSSFeed struct {
	Channel RSSChannel `xml:"channel"`
}

type RSSChannel struct {
	Title       string    `xml:"title"`
	Description string    `xml:"description"`
	Link        string    `xml:"link"`
	Item        []RSSItem `xml:"item"`
}

type RSSItem struct {
	Title       string `xml:"title"`
	Description string `xml:"description"`
	Link        string `xml:"link"`
	PubDate     string `xml:"pubDate"`
}

func (c *commands) run(s *state, cmd command) error {
	handler, exists := c.handlers[cmd.name]
	if !exists {
		fmt.Println("command does not exist")
		os.Exit(1)
	}

	return handler(s, cmd)
}

func (c *commands) register(name string, f func(*state, command) error) {
	c.handlers[name] = f
}

func fetchFeed(ctx context.Context, feedURL string) (*RSSFeed, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		feedURL,
		nil,
	)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "gator")

	client := &http.Client{}

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var feed RSSFeed

	err = xml.Unmarshal(data, &feed)
	if err != nil {
		return nil, err
	}

	feed.Channel.Title =
		html.UnescapeString(feed.Channel.Title)

	feed.Channel.Description =
		html.UnescapeString(feed.Channel.Description)

	for i := range feed.Channel.Item {
		feed.Channel.Item[i].Title =
			html.UnescapeString(feed.Channel.Item[i].Title)

		feed.Channel.Item[i].Description =
			html.UnescapeString(feed.Channel.Item[i].Description)
	}

	return &feed, nil
}

func handlerLogin(s *state, cmd command) error {
	if len(cmd.args) == 0 {
		fmt.Println("username is required")
		os.Exit(1)
	}

	username := cmd.args[0]

	_, err := s.db.GetUser(
		context.Background(),
		username,
	)
	if err != nil {
		fmt.Printf("user %s does not exist\n", username)
		os.Exit(1)
	}

	err = s.cfg.SetUser(username)
	if err != nil {
		return err
	}

	fmt.Printf("User has been set to %s\n", username)

	return nil
}

func handlerRegister(s *state, cmd command) error {
	if len(cmd.args) == 0 {
		fmt.Println("username is required")
		os.Exit(1)
	}

	name := cmd.args[0]
	now := time.Now()

	_, err := s.db.CreateUser(
		context.Background(),
		database.CreateUserParams{
			ID:        uuid.New(),
			CreatedAt: now,
			UpdatedAt: now,
			Name:      name,
		},
	)

	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			fmt.Println("user already exists")
			os.Exit(1)
		}

		return err
	}

	err = s.cfg.SetUser(name)
	if err != nil {
		return err
	}

	fmt.Printf("User %s created successfully\n", name)

	return nil
}

func handlerReset(s *state, cmd command) error {
	err := s.db.DeleteUsers(context.Background())
	if err != nil {
		return err
	}

	fmt.Println("All users deleted successfully")

	return nil
}

func handlerUsers(s *state, cmd command) error {
	users, err := s.db.GetUsers(context.Background())
	if err != nil {
		return err
	}

	for _, user := range users {
		if user.Name == s.cfg.CurrentUserName {
			fmt.Printf("* %s (current)\n", user.Name)
		} else {
			fmt.Printf("* %s\n", user.Name)
		}
	}

	return nil
}

func handlerAgg(s *state, cmd command) error {
	if len(cmd.args) != 1 {
		return fmt.Errorf("usage: agg <time_between_reqs>")
	}

	timeBetweenRequests, err :=
		time.ParseDuration(cmd.args[0])
	if err != nil {
		return err
	}

	fmt.Printf(
		"Collecting feeds every %v\n",
		timeBetweenRequests,
	)

	ticker := time.NewTicker(timeBetweenRequests)
	defer ticker.Stop()

	for ; ; <-ticker.C {
		err := scrapeFeeds(s)
		if err != nil {
			fmt.Println(err)
		}
	}

	return nil
}

func handlerAddFeed(
	s *state,
	cmd command,
	user database.User,
	) error {
	if len(cmd.args) < 2 {
		return fmt.Errorf("usage: addfeed <name> <url>")
	}

	name := cmd.args[0]
	url := cmd.args[1]

	user, err := s.db.GetUser(
		context.Background(),
		s.cfg.CurrentUserName,
	)
	if err != nil {
		return err
	}

	now := time.Now()

	feed, err := s.db.CreateFeed(
		context.Background(),
		database.CreateFeedParams{
			ID:        uuid.New(),
			CreatedAt: now,
			UpdatedAt: now,
			Name:      name,
			Url:       url,
			UserID:    user.ID,
		},
	)
	if err != nil {
		return err
	}

	_, err = s.db.CreateFeedFollow(
		context.Background(),
		database.CreateFeedFollowParams{
			ID:        uuid.New(),
			CreatedAt: now,
			UpdatedAt: now,
			UserID:    user.ID,
			FeedID:    feed.ID,
		},
	)
	if err != nil {
		return err
	}

	fmt.Printf("%+v\n", feed)
	
	return nil
}

func handlerFeeds(s *state, cmd command) error {
	feeds, err := s.db.GetFeeds(context.Background())
	if err != nil {
		return err
	}

	for _, feed := range feeds {
		fmt.Printf("Name: %s\n", feed.Name)
		fmt.Printf("URL: %s\n", feed.Url)
		fmt.Printf("User: %s\n", feed.UserName)
		fmt.Println()
	}
	
	return nil
}

func handlerFollow(
	s *state,
	cmd command,
	user database.User,
) error {
	if len(cmd.args) < 1 {
		return fmt.Errorf("usage: follow <url>")
	}

	feedURL := cmd.args[0]

	user, err := s.db.GetUser(
		context.Background(),
		s.cfg.CurrentUserName,
	)
	if err != nil {
		return err
	}

	feed, err := s.db.GetFeedByURL(
		context.Background(),
		feedURL,
	)
	if err != nil {
		return err
	}

	now := time.Now()

	ff, err := s.db.CreateFeedFollow(
		context.Background(),
		database.CreateFeedFollowParams{
			ID:        uuid.New(),
			CreatedAt: now,
			UpdatedAt: now,
			UserID:    user.ID,
			FeedID:    feed.ID,
		},
	)
	if err != nil {
		return err
	}

	fmt.Printf("%s is now following %s\n",
		ff.UserName,
		ff.FeedName,
	)

	return nil
}

func handlerFollowing(
	s *state,
	cmd command,
	user database.User,
) error {
	user, err := s.db.GetUser(
		context.Background(),
		s.cfg.CurrentUserName,
	)
	if err != nil {
		return err
	}

	follows, err := s.db.GetFeedFollowsForUser(
		context.Background(),
		user.ID,
	)
	if err != nil {
		return err
	}

	for _, follow := range follows {
		fmt.Println(follow.FeedName)
	}

	return nil
}

func handlerUnfollow(s *state, cmd command, user database.User) error {
	if len(cmd.args) < 1 {
		return fmt.Errorf("usage: unfollow <url>")
	}

	feedURL := cmd.args[0]

	feed, err := s.db.GetFeedByURL(
		context.Background(),
		feedURL,
	)
	if err != nil {
		return err
	}

	err = s.db.DeleteFeedFollow(
		context.Background(),
		database.DeleteFeedFollowParams{
			UserID: user.ID,
			FeedID: feed.ID,
		},
	)
	if err != nil {
		return err
	}

	fmt.Printf("Unfollowed %s\n", feed.Name)

	return nil
}

func scrapeFeeds(s *state) error {
	feed, err := s.db.GetNextFeedToFetch(
		context.Background(),
	)
	if err != nil {
		return err
	}

	err = s.db.MarkFeedFetched(
		context.Background(),
		feed.ID,
	)
	if err != nil {
		return err
	}

	rssFeed, err := fetchFeed(
		context.Background(),
		feed.Url,
	)
	if err != nil {
		return err
	}

	fmt.Printf("Fetching feed: %s\n", feed.Name)

	for _, item := range rssFeed.Channel.Item {
		_, err := s.db.CreatePost(
			context.Background(),
			database.CreatePostParams{
				ID:          uuid.New(),
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
				Title:       item.Title,
				Url:         item.Link,
				Description: item.Description,
				PublishedAt: publishedAt,
				FeedID:      feed.ID,
			},
		)

		if err != nil {
			if strings.Contains(err.Error(), "duplicate key") {
				continue
			}

			fmt.Println(err)
		}
	}

	return nil
}

func middlewareLoggedIn(
	handler func(s *state, cmd command, user database.User) error,
) func(*state, command) error {
	return func(s *state, cmd command) error {
		user, err := s.db.GetUser(
			context.Background(),
			s.cfg.CurrentUserName,
		)
		if err != nil {
			return err
		}

		return handler(s, cmd, user)
	}
}

func main() {
	cfg, err := config.Read()
	if err != nil {
		fmt.Println("Error reading config:", err)
		return
	}

	db, err := sql.Open("postgres", cfg.DBURL)
	if err != nil {
		fmt.Println("Error opening database:", err)
		return
	}

	dbQueries := database.New(db)

	s := state{
		db:  dbQueries,
		cfg: &cfg,
	}

	cmds := commands{
		handlers: make(map[string]func(*state, command) error),
	}

	cmds.register("login", handlerLogin)
	cmds.register("register", handlerRegister)
	cmds.register("reset", handlerReset)
	cmds.register("users", handlerUsers)
	cmds.register("agg", handlerAgg)
	cmds.register("feeds", handlerFeeds)
	cmds.register(
		"addfeed",
		middlewareLoggedIn(handlerAddFeed),
	)
	cmds.register(
		"follow",
		middlewareLoggedIn(handlerFollow),
	)
	cmds.register(
		"following",
		middlewareLoggedIn(handlerFollowing),
	)
	cmds.register(
		"unfollow",
		middlewareLoggedIn(handlerUnfollow),
	)

	if len(os.Args) < 2 {
		fmt.Println("not enough arguments")
		os.Exit(1)
	}

	cmd := command{
		name: os.Args[1],
		args: os.Args[2:],
	}

	err = cmds.run(&s, cmd)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
