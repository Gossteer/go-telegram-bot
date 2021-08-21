package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type bResponse struct {
	Symbol string  `json:"symbol"`
	Price  float64 `json:"price,string"`
}

type botSendStruct struct {
	bot     *tgbotapi.BotAPI
	chat_id int64
	message string
}

type wallet map[string]float64

var db = map[int]wallet{}

func main() {
	bot, err := tgbotapi.NewBotAPI(getToken())
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, _ := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		commands := strings.Split(update.Message.Text, " ")
		len_commands := len(commands)
		user_id := update.Message.From.ID
		botSendS := botSendStruct{bot, update.Message.Chat.ID, "Я бот Антон"}

		if len_commands > 0 {
			switch strings.ToLower(commands[0]) {
			case "add":
				if !chekLenCommand(len_commands, 3) {
					botSend(setMessageScruct(&botSendS, "Неверные аргументы"))

					continue
				}

				if err := setPrice(user_id, commands); err != nil {
					botSend(setMessageScruct(&botSendS, err.Error()))

					continue
				}
			case "sub":
				if !chekLenCommand(len_commands, 3) {
					botSend(setMessageScruct(&botSendS, "Неверные аргументы"))

					continue
				}

				if err := setPrice(user_id, commands); err != nil {
					botSend(setMessageScruct(&botSendS, err.Error()))

					continue
				}
			case "del":
				if !chekLenCommand(len_commands, 2) {
					botSend(setMessageScruct(&botSendS, "Неверные аргументы"))

					continue
				}

				delete(db[user_id], commands[1])
			case "show":
				if err := sendShowPrice(botSendS, []string{"RUB", "USDT"}, user_id); err != nil {
					botSend(setMessageScruct(&botSendS, err.Error()))
				}
			case "showrub":
				if err := sendShowPrice(botSendS, []string{"RUB"}, user_id); err != nil {
					botSend(setMessageScruct(&botSendS, err.Error()))
				}
			case "showusdt":
				if err := sendShowPrice(botSendS, []string{"USDT"}, user_id); err != nil {
					botSend(setMessageScruct(&botSendS, err.Error()))
				}
			case "/help":
				botSend(setMessageScruct(&botSendS, "Список команд (вводи без скобок, можно в нижнем регистре):\nДля добавления валюты - ADD [название валюты] [сумма (можно десятичное используй точку '.')]\nДля снятия валюты - SUB [название валюты] [сумма (можно десятичное используй точку '.')]\nДля удаления валюты - DEL [название валюты]\nДля демонстрации всех валют в кошельке - SHOW(RUB/USDT)\nПримеры команд: \nADD BTC 0.15\nADD ETH 3.1225\nADD XRP 12.1\nSUB BTC 0.09\nDEL BTC\nSHOW"))
			case "/start":
				botSend(setMessageScruct(&botSendS, "Привет, я тестовый бот по криптовалютному кошельку. Все подробности можно узнать через /help")) //fix
			default:
				botSend(setMessageScruct(&botSendS, "Я тебя не понял, повторись: "+commands[0]))
			}
		} else {
			botSend(setMessageScruct(&botSendS, "Неверные аргументы"))
		}

		// botSend(setMessageScruct(&botSendS, update.CallbackQuery.Data))
		log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)
	}
}

func setPrice(user_id int, commands []string) error {
	money, err := strconv.ParseFloat(commands[2], 64)
	if err != nil {
		return errors.New("ошибка преобразования цена")
	}

	if _, ok := db[user_id]; !ok {
		db[user_id] = make(wallet)
	}

	db[user_id][commands[1]] += money

	return nil
}

func sendShowPrice(botSendS botSendStruct, symbols_to []string, user_id int) error {
	resp := ""
	for key, value := range db[user_id] {

		for _, symbol_to := range symbols_to {
			rub_price, err := getPrice(key, symbol_to)

			if err != nil {
				botSend(setMessageScruct(&botSendS, err.Error()))

				return err
			}

			resp += fmt.Sprintf("%s (%s): %.2f\n", key, symbol_to, value*rub_price)
		}
	}
	botSend(setMessageScruct(&botSendS, resp))

	return nil
}

func getToken() string {
	return ""
}

func setMessageScruct(botSendS *botSendStruct, message string) botSendStruct {
	botSendS.message = message

	return *botSendS
}

func botSend(botSendS botSendStruct) (tgbotapi.Message, error) {
	lol := tgbotapi.NewMessage(botSendS.chat_id, botSendS.message)

	return botSendS.bot.Send(lol)
}

func chekLenCommand(command int, need_len int) bool {
	return (command == need_len)
}

func getPrice(symbol_in string, symbol_to string) (float64, error) {
	url := fmt.Sprintf("https://api.binance.com/api/v3/ticker/price?symbol=%s%s", symbol_in, symbol_to)
	resp, err := http.Get(url)

	if err != nil {
		return 0, err
	}

	var bRes bResponse

	err = json.NewDecoder(resp.Body).Decode(&bRes)
	if err != nil {
		return 0, err
	}

	if bRes.Symbol == "" {
		return 0, errors.New("неверная валюта")
	}

	return bRes.Price, nil
}
