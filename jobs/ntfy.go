package jobs

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

func SendMessage(content string) error {
	req, err := http.NewRequest("POST", "https://notifications.mobilecraft.io/reddit_mentions", strings.NewReader(content))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Authorization", "Bearer tk_ml841hx0boanq7n3lzsc9dpfgf6qu")

	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		return err
	}

	defer func(body io.ReadCloser) {
		if err := body.Close(); err != nil {
			log.Println("failed to close response body")
		}
	}(response.Body)

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to send successful request. Status was %q", response.Status)
	}
	return nil
}
