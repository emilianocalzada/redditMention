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

func main() {
	app := pocketbase.New()

	// Load env variables
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	everyMinutes := os.Getenv("EVERY_MINUTES")
	if everyMinutes == "" {
		log.Fatal("EVERY_MINUTES is not set")
	}

	// hour, day, week, month, year or all
	fromTime := os.Getenv("FROM_TIME")
	if fromTime == "" {
		log.Fatal("FROM_TIME is not set")
	}

	// Schedule the job
	cronExpression := fmt.Sprintf("*/%s * * * *", everyMinutes)
	app.Cron().Add("redditPosts", cronExpression, func() {
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

		posts, err := jobs.GetRedditPosts(keywordsKeys, 1, fromTime, app)
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
			err := jobs.SendMessage(message)
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
