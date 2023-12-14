package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	telebot "gopkg.in/telebot.v3"
)

func main() {
	bot, err := telebot.NewBot(telebot.Settings{
		Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
		Token:  "",
	})
	fmt.Println("Bot Started")
	if err != nil {
		fmt.Println(err)
		return
	}

	bot.Handle(telebot.OnDocument, func(m telebot.Context) error {
		url := UrlGenerator(m.Message(), bot)
		go downloadMedia(bot, m.Message(), url, filepath.Base(url))
		fmt.Println(url, m)
		return nil
	})
	bot.Handle(telebot.OnVideo, func(m telebot.Context) error {
		url := UrlGenerator(m.Message(), bot)
		go downloadMedia(bot, m.Message(), url, filepath.Base(url))
		fmt.Println(url)
		return nil

	})
	bot.Handle(telebot.OnPhoto, func(m telebot.Context) error {
		url := UrlGenerator(m.Message(), bot)
		go downloadMedia(bot, m.Message(), url, filepath.Base(url))
		fmt.Println(url)
		return nil

	})
	bot.Handle(telebot.OnAudio, func(m telebot.Context) error {
		url := UrlGenerator(m.Message(), bot)
		go downloadMedia(bot, m.Message(), url, filepath.Base(url))
		fmt.Println(url)
		return nil

	})

	bot.Start()
}

func UrlGenerator(m *telebot.Message, bot *telebot.Bot) string {
	g, _ := bot.FileByID(m.Photo.FileID)
	return fmt.Sprintf("https://api.telegram.org/file/bot%s/%s",
		bot.Token, g.FilePath)
}

func downloadMedia(bot *telebot.Bot, msg *telebot.Message, url string, fName string) {

	resp, err := http.Get(url)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer resp.Body.Close()

	out, err := os.Create(fName) // Change the filename and extension as needed
	if err != nil {
		fmt.Println(err)
		return
	}
	defer out.Close()

	bar, err := bot.Send(msg.Sender, "Downloading...")

	done := make(chan int64)

	go func() {
		defer close(done)
		n, err := io.Copy(out, io.TeeReader(resp.Body, &progressReader{total: resp.ContentLength, bar: bar, done: done, bot: bot}))
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Printf("Downloaded %d bytes\n", n)
	}()

	<-done
	bot.Edit(bar, "Download complete!")
}

type progressReader struct {
	total   int64
	bar     *telebot.Message
	done    chan<- int64
	bot     *telebot.Bot
	current int64
}

func (pr *progressReader) Write(p []byte) (int, error) {
	n := len(p)
	pr.current += int64(n)
	text := fmt.Sprintf("Downloading... %d%%", pr.current*100/pr.total)
	pr.bot.Edit(pr.bar, text)
	if pr.total == 100 {
		pr.done <- 1
	}
	return n, nil
}
