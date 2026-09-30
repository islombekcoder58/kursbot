module kursbot

go 1.22

require gopkg.in/telebot.v3 v3.3.8

require (
	github.com/deckarep/golang-set/v2 v2.8.0 // indirect
	github.com/go-stack/stack v1.8.1 // indirect
	github.com/mxschmitt/playwright-go v0.6201.1 // indirect
)

replace gopkg.in/telebot.v3 => ./local/gopkg.in/telebot.v3@v3.3.8
