package updatedetector

import (
	"log"
	"net/http"
	"os"
)

func init() {
	bot, err := NewUpdateDetectorBot(
		os.Getenv("CHANNEL_SECRET"),
		os.Getenv("CHANNEL_TOKEN"),
	)
	if err != nil {
		log.Fatal(err)
	}
	http.HandleFunc("/callback"+os.Getenv("PATH_SUFFIX"), bot.WebHook)
	http.HandleFunc("/cron"+os.Getenv("PATH_SUFFIX"), bot.CronHook)

}
