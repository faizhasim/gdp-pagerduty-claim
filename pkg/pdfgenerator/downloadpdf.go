package pdfgenerator

import (
	"fmt"
	"io"
	"net/http"
	"os"
)

func DownloadPdfTemplate() (string, error) {
	filepath := os.TempDir() + "/template.png"
	url := "https://user-images.githubusercontent.com/898384/54095034-e4fc0380-43df-11e9-8ad2-c263b3a71c71.png"

	if _, err := os.Stat(filepath); err == nil {
		return filepath, nil
	}

	out, err := os.Create(filepath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status: %s", resp.Status)
	}

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", err
	}

	return filepath, nil
}
