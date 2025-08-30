package jobs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/pocketbase/pocketbase"
)

// Request structures
type BulkRequest struct {
	Password string   `json:"password"`
	Action   string   `json:"action"`
	Data     BulkData `json:"data"`
}

type BulkData struct {
	Parser       string    `json:"parser"`
	Preset       string    `json:"preset"`
	ConfigPreset string    `json:"configPreset"`
	Threads      int       `json:"threads"`
	RawResults   int       `json:"rawResults"`
	Queries      []string  `json:"queries"`
	Options      []Options `json:"options"`
	DoLog        int       `json:"doLog"`
}

type Options struct {
	Type  string `json:"type"`
	Id    string `json:"id"`
	Value string `json:"value"`
}

// Response structures
type RedditResponse struct {
	Success int `json:"success"`
	Data    struct {
		ResultString string `json:"resultString"`
	} `json:"data"`
}

type RedditPost struct {
	Query     string `json:"query"`
	Title     string `json:"title"`
	Link      string `json:"link"`
	Subreddit string `json:"subreddit"`
	Flair     string `json:"flair"`
}

func parseRedditResponse(body []byte, app *pocketbase.PocketBase) ([]RedditPost, error) {
	// First parse the outer response structure
	var response RedditResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	// The ResultString contains escaped JSON, so we need to parse it separately
	// First unescape the JSON string
	resultString := response.Data.ResultString

	// Parse the result string into array of RedditPost
	var posts []RedditPost

	// Split result string into lines
	lines := strings.Split(resultString, "\n")

	// Parse each line into a RedditPost
	for _, line := range lines {
		if line == "" {
			continue
		}

		// 1. Remove extra double quotes from: {"query":""email validator"","title"
		// it should be: {"query":"email validator","title"
		line = strings.Replace(line, "{\"query\":\"\"", "{\"query\":\"", 1)
		line = strings.Replace(line, "\"\",\"title\"", "\",\"title\"", 1)

		// 2. Extract the text between START_TITLE and END_TITLE
		startIndex := strings.Index(line, "START_TITLE ") + len("START_TITLE ")
		endIndex := strings.Index(line, " END_TITLE")
		if startIndex == -1 || endIndex == -1 {
			app.Logger().Error("Missing START_TITLE or END_TITLE markers in line: " + line)
			continue
		}

		originalTitle := line[startIndex:endIndex]

		// 3. Create a new title with single quotes instead of double quotes
		newTitle := strings.ReplaceAll(originalTitle, "\"", "'")

		// 4. Replace the entire START_TITLE...END_TITLE section with the processed title
		processedLine := line[:strings.Index(line, "START_TITLE")] +
			newTitle +
			line[strings.Index(line, "END_TITLE")+len("END_TITLE"):]

		// 5. Now parse the processed JSON
		var post RedditPost
		if err := json.Unmarshal([]byte(processedLine), &post); err != nil {
			app.Logger().Error("Failed to parse line: " + line)
			continue
		}
		posts = append(posts, post)
	}

	return posts, nil
}

func GetRedditPosts(queries []string, pageCount int, time string, app *pocketbase.PocketBase, endpoint, password string) ([]RedditPost, error) {
	// Create the request payload
	request := BulkRequest{
		Password: password,
		Action:   "bulkRequest",
		Data: BulkData{
			Parser:       "Reddit::Posts",
			Preset:       "mojoproxy",
			ConfigPreset: "default",
			Threads:      50,
			RawResults:   0,
			Queries:      queries,
			Options: []Options{
				{
					Type:  "override",
					Id:    "pagecount",
					Value: strconv.Itoa(pageCount),
				},
				{
					Type:  "override",
					Id:    "sort",
					Value: "relevance",
				},
				{
					Type:  "override",
					Id:    "time",
					Value: time,
				},
				{
					Type:  "override",
					Id:    "formatresult",
					Value: "[%\nposts.format('{\"query\":\"$query\",\"title\":\"START_TITLE $title END_TITLE\",\"link\":\"$link\",\"subreddit\":\"$subreddit\",\"flair\":\"$flair\"}\\n')\n%]",
				},
				{
					Type:  "override",
					Id:    "queryformat",
					Value: "\"$query\"",
				},
			},
			DoLog: 0,
		},
	}

	// Convert the request to JSON
	jsonData, err := json.Marshal(request)
	if err != nil {
		fmt.Printf("Error marshaling JSON: %v\n", err)
		return nil, err
	}

	// Create the HTTP request
	// Replace "YOUR_API_ENDPOINT" with the actual API endpoint
	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return nil, err
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")

	// Create HTTP client and send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error sending request: %v\n", err)
		return nil, err
	}
	defer resp.Body.Close()

	// Read the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		return nil, err
	}

	posts, err := parseRedditResponse(body, app)
	if err != nil {
		// Handle error
		fmt.Printf("Error parsing response: %v\n", err)
		return nil, err
	}

	return posts, nil
}
