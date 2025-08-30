package jobs

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

func SendMessage(endpoint, token, content string) error {
	req, err := http.NewRequest("POST", endpoint, strings.NewReader(content))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Authorization", "Bearer "+token)

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
