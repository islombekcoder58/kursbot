package main

import (
	"context"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	tele "gopkg.in/telebot.v3"
)

const welcome = `🤖 KursBot

Valyuta kurslari va konvertatsiya botiga xush kelibsiz!

Kerakli bo‘limni tanlang 👇`

const helpText = `ℹ️ Yordam

💵 Kurslar
Bugungi USD, EUR va RUB kurslarini ko‘rsatadi.

🔄 Konvertatsiya
Valyutalarni tez va qulay hisoblash imkonini beradi.

Misollar:
• 100 USD → UZS
• 500 EUR → UZS
• 1 000 000 UZS → USD

Eski format ham ishlaydi:
• 500 usd
• 2 mln so'm
• 100 yevro
• 1500 rubl

📌 Kurslar O‘zbekiston Respublikasi Markaziy banki ma'lumotlari asosida olinadi.`

type conversionState struct {
	From string
	To   string
}

type stateStore struct {
	mu     sync.RWMutex
	states map[int64]conversionState
}

func newStateStore() *stateStore {
	return &stateStore{
		states: make(map[int64]conversionState),
	}
}

func (s *stateStore) set(userID int64, state conversionState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.states[userID] = state
}

func (s *stateStore) get(userID int64) (conversionState, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state, ok := s.states[userID]
	return state, ok
}

func (s *stateStore) delete(userID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.states, userID)
}

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
	states := newStateStore()

	// Asosiy menyu
	menu := &tele.ReplyMarkup{
		ResizeKeyboard:  true,
		OneTimeKeyboard: false,
	}

	menu.Reply(
		menu.Row(
			menu.Text("💵 Kurslar"),
			menu.Text("🔄 Konvertatsiya"),
		),
		menu.Row(
			menu.Text("ℹ️ Yordam"),
		),
	)

	// /start
	bot.Handle("/start", func(c tele.Context) error {
		if c.Sender() != nil {
			states.delete(c.Sender().ID)
		}

		return c.Send(welcome, menu)
	})

	// 💵 Kurslar
	bot.Handle("💵 Kurslar", func(c tele.Context) error {
		return sendRates(c, store)
	})

	// /kurs ham saqlanadi
	bot.Handle("/kurs", func(c tele.Context) error {
		return sendRates(c, store)
	})

	// 🔄 Konvertatsiya
	bot.Handle("🔄 Konvertatsiya", func(c tele.Context) error {
		if c.Sender() != nil {
			states.delete(c.Sender().ID)
		}

		return c.Send(
			"🔄 Konvertatsiya\n\nAvval valyutalarni tanlang:",
			conversionKeyboard(),
		)
	})

	// ℹ️ Yordam
	bot.Handle("ℹ️ Yordam", func(c tele.Context) error {
		return c.Send(helpText, menu)
	})

	// Inline callbacklar
	bot.Handle(&tele.Btn{
		Unique: "conv_usd_uzs",
		Text:   "💵 USD → UZS",
	}, func(c tele.Context) error {
		return startConversion(c, states, "USD", "UZS")
	})

	bot.Handle(&tele.Btn{
		Unique: "conv_eur_uzs",
		Text:   "💶 EUR → UZS",
	}, func(c tele.Context) error {
		return startConversion(c, states, "EUR", "UZS")
	})

	bot.Handle(&tele.Btn{
		Unique: "conv_rub_uzs",
		Text:   "🇷🇺 RUB → UZS",
	}, func(c tele.Context) error {
		return startConversion(c, states, "RUB", "UZS")
	})

	bot.Handle(&tele.Btn{
		Unique: "conv_uzs_usd",
		Text:   "🇺🇿 UZS → USD",
	}, func(c tele.Context) error {
		return startConversion(c, states, "UZS", "USD")
	})

	bot.Handle(&tele.Btn{
		Unique: "conv_uzs_eur",
		Text:   "🇺🇿 UZS → EUR",
	}, func(c tele.Context) error {
		return startConversion(c, states, "UZS", "EUR")
	})

	bot.Handle(&tele.Btn{
		Unique: "conv_uzs_rub",
		Text:   "🇺🇿 UZS → RUB",
	}, func(c tele.Context) error {
		return startConversion(c, states, "UZS", "RUB")
	})

	bot.Handle(&tele.Btn{
		Unique: "conv_back",
		Text:   "⬅️ Orqaga",
	}, func(c tele.Context) error {
		if c.Sender() != nil {
			states.delete(c.Sender().ID)
		}

		return c.Send(welcome, menu)
	})

	// Oddiy text va konvertatsiya
	bot.Handle(tele.OnText, func(c tele.Context) error {
		text := strings.TrimSpace(c.Text())

		// Menyu tugmalari yana shu handlerga tushib qolsa,
		// xatolik xabarini chiqarmaymiz.
		switch text {
		case "💵 Kurslar", "🔄 Konvertatsiya", "ℹ️ Yordam":
			return nil
		}

		// Inline orqali valyuta juftligi tanlangan bo‘lsa
		if c.Sender() != nil {
			if state, ok := states.get(c.Sender().ID); ok {
				amount, ok := ParseAmount(text)
				if !ok {
					return c.Send(
						"❌ Summani tushunmadim.\n\nMasalan:\n100\n500.50\n1 000",
					)
				}

				ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
				defer cancel()

				rates, err := store.Get(ctx)
				if err != nil {
					log.Println("kurs xatosi:", err)
					return c.Send("⚠️ Kursni olib bo‘lmadi. Birozdan keyin urinib ko‘ring.")
				}

				states.delete(c.Sender().ID)

				return c.Send(
					ConvertPair(amount, state.From, state.To, rates),
					menu,
				)
			}
		}

		// Masalan: 100 USD → UZS
		if q, ok := ParseConversionQuery(text); ok {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			rates, err := store.Get(ctx)
			if err != nil {
				log.Println("kurs xatosi:", err)
				return c.Send("⚠️ Kursni olib bo‘lmadi. Birozdan keyin urinib ko‘ring.")
			}

			return c.Send(ConvertPair(q.Amount, q.From, q.To, rates), menu)
		}

		// Eski formatlar:
		// 500 usd
		// 2 mln so'm
		// 100 yevro
		q, ok := ParseQuery(text)
		if !ok {
			return c.Send(
				"🤔 Tushunmadim.\n\n" +
					"Masalan:\n" +
					"• 100 USD → UZS\n" +
					"• 500 EUR → UZS\n" +
					"• 500 usd\n" +
					"• 2 mln so'm",
			)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		rates, err := store.Get(ctx)
		if err != nil {
			log.Println("kurs xatosi:", err)
			return c.Send("⚠️ Kursni olib bo‘lmadi. Birozdan keyin urinib ko‘ring.")
		}

		return c.Send(Convert(q, rates), menu)
	})

	log.Println("Bot ishga tushdi")
	bot.Start()
}

func sendRates(c tele.Context, store *RateStore) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	rates, err := store.Get(ctx)
	if err != nil {
		log.Println("kurs xatosi:", err)
		return c.Send("⚠️ Kursni olib bo‘lmadi. Birozdan keyin urinib ko‘ring.")
	}

	return c.Send(RatesText(rates))
}

func conversionKeyboard() *tele.ReplyMarkup {
	kb := &tele.ReplyMarkup{}

	kb.Inline(
		kb.Row(
			tele.Btn{
				Unique: "conv_usd_uzs",
				Text:   "💵 USD → UZS",
			},
			tele.Btn{
				Unique: "conv_eur_uzs",
				Text:   "💶 EUR → UZS",
			},
		),
		kb.Row(
			tele.Btn{
				Unique: "conv_rub_uzs",
				Text:   "🇷🇺 RUB → UZS",
			},
		),
		kb.Row(
			tele.Btn{
				Unique: "conv_uzs_usd",
				Text:   "🇺🇿 UZS → USD",
			},
			tele.Btn{
				Unique: "conv_uzs_eur",
				Text:   "🇺🇿 UZS → EUR",
			},
		),
		kb.Row(
			tele.Btn{
				Unique: "conv_uzs_rub",
				Text:   "🇺🇿 UZS → RUB",
			},
		),
		kb.Row(
			tele.Btn{
				Unique: "conv_back",
				Text:   "⬅️ Orqaga",
			},
		),
	)

	return kb
}

func startConversion(
	c tele.Context,
	states *stateStore,
	from string,
	to string,
) error {
	if c.Sender() == nil {
		return nil
	}

	// Telegramdagi loading holatini olib tashlash.
	_ = c.Respond()

	states.set(c.Sender().ID, conversionState{
		From: from,
		To:   to,
	})

	return c.Send(
		"🔄 " + from + " → " + to + "\n\n" +
			"Summani yuboring.\n\n" +
			"Masalan: 100",
	)
}
