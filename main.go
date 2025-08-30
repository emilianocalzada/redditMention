package main

import (
	"fmt"
	"log"
	"os"
	"redditMention/jobs"

	"github.com/joho/godotenv"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

type Keyword struct {
	Id       string `db:"id" json:"id"`
	Keyword  string `db:"keyword" json:"keyword"`
	Track_it string `db:"track_it" json:"track_it"`
	Created  string `db:"created" json:"created"`
}

func init() {
	// Only load .env if it exists (e.g. local dev)
	if _, err := os.Stat(".env"); err == nil {
		if err := godotenv.Load(); err != nil {
			log.Println("Warning: could not load .env file")
		}
	}
}

func main() {
	app := pocketbase.New()

	// Check for new reddit posts every X minutes
	everyMinutes := os.Getenv("EVERY_MINUTES")
	if everyMinutes == "" {
		log.Fatal("EVERY_MINUTES is not set")
	}

	// hour, day, week, month, year or all
	fromTime := os.Getenv("FROM_TIME")
	if fromTime == "" {
		log.Fatal("FROM_TIME is not set")
	}

	// Number of pages to search
	pageCount := os.Getenv("PAGE_COUNT")
	if pageCount == "" {
		log.Fatal("PAGE_COUNT is not set")
	}

	// ntfy endpoint and token
	ntfyEndpoint := os.Getenv("NTFY_ENDPOINT")
	if ntfyEndpoint == "" {
		log.Fatal("NTFY_ENDPOINT is not set")
	}

	ntfyToken := os.Getenv("NTFY_TOKEN")
	if ntfyToken == "" {
		log.Fatal("NTFY_TOKEN is not set")
	}

	// A-parser endpoint and password
	aparserEndpoint := os.Getenv("APARSER_ENDPOINT")
	if aparserEndpoint == "" {
		log.Fatal("APARSER_ENDPOINT is not set")
	}
	aparserPassword := os.Getenv("APARSER_PASSWORD")
	if aparserPassword == "" {
		log.Fatal("APARSER_PASSWORD is not set")
	}

	// Schedule the job
	cronExpression := fmt.Sprintf("*/%s * * * *", everyMinutes)
	app.Cron().MustAdd("redditPosts", cronExpression, func() {
		app.Logger().Info("Running redditPosts job")

		// get keywords
		keywordRecords := []Keyword{}
		err := app.DB().
			NewQuery("SELECT id, keyword, track_it, created FROM keywords WHERE track_it = true").
			All(&keywordRecords)

		if err != nil {
			log.Println(err)
			return
		}

		keywords := map[string]string{}
		for _, keyword := range keywordRecords {
			keywords[keyword.Keyword] = keyword.Id
		}

		if len(keywords) == 0 {
			app.Logger().Info("No keywords found")
			return
		}

		app.Logger().Info(fmt.Sprintf("Keywords: %v", keywords))

		// get keywords keys array of string
		keywordsKeys := []string{}
		for keyword := range keywords {
			keywordsKeys = append(keywordsKeys, keyword)
		}

		posts, err := jobs.GetRedditPosts(keywordsKeys, pageCount, fromTime, app, aparserEndpoint, aparserPassword)
		if err != nil {
			app.Logger().Error(fmt.Sprintf("Failed to get reddit posts: %v", err))
			return
		}

		collection, err := app.FindCollectionByNameOrId("posts")
		if err != nil {
			app.Logger().Error(fmt.Sprintf("Collection not found: %v", err))
			return
		}

		newPostsCount := 0
		for _, post := range posts {
			record := core.NewRecord(collection)
			record.Load(map[string]any{
				"keyword":   keywords[post.Query],
				"title":     post.Title,
				"url":       post.Link,
				"subreddit": post.Subreddit,
				"flair":     post.Flair,
			})
			if err := app.Save(record); err != nil {
				app.Logger().Error(fmt.Sprintf("Failed to save record: %v", err))
				continue
			}
			newPostsCount++
		}

		// Notify to NTFY
		message := fmt.Sprintf("Found %d new relevant posts on reddit", newPostsCount)
		if newPostsCount > 0 {
			err := jobs.SendMessage(ntfyEndpoint, ntfyToken, message)
			if err != nil {
				app.Logger().Error(fmt.Sprintf("Failed to send ntfy message: %v", err))
			}
		}

		app.Logger().Info(fmt.Sprintf("Saved %d posts", newPostsCount))
	})

	// Start app
	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
