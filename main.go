package main

import (
	"context"
	"log"
	"os"
	"time"

	tele "gopkg.in/telebot.v3"
)

const welcome = `Assalomu alaykum! 👋

Men valyuta kursi va konvertor botman.

/kurs - bugungi kurs (Markaziy bank)

Yoki oddiy yozing, men hisoblab beraman:
• 500 usd
• 2 mln so'm
• 100 yevro
• 1500 rubl`

func main() {
	token := os.Getenv("BOT_TOKEN")
	if token == "" {
		log.Fatal("BOT_TOKEN o'rnatilmagan")
	}

	bot, err := tele.NewBot(tele.Settings{
		Token:  token,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil {
		log.Fatal(err)
	}

	store := NewRateStore()

	bot.Handle("/start", func(c tele.Context) error {
		return c.Send(welcome)
	})

	bot.Handle("/kurs", func(c tele.Context) error {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		rates, err := store.Get(ctx)
		if err != nil {
			log.Println("kurs xatosi:", err)
			return c.Send("Kursni olib bo'lmadi, birozdan keyin urinib ko'ring.")
		}
		return c.Send(RatesText(rates))
	})

	bot.Handle(tele.OnText, func(c tele.Context) error {
		q, ok := ParseQuery(c.Text())
		if !ok {
			return c.Send("Tushunmadim 🤔\nMasalan: 500 usd yoki 2 mln so'm")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		rates, err := store.Get(ctx)
		if err != nil {
			log.Println("kurs xatosi:", err)
			return c.Send("Kursni olib bo'lmadi, birozdan keyin urinib ko'ring.")
		}
		return c.Send(Convert(q, rates))
	})

	log.Println("Bot ishga tushdi")
	bot.Start()
}