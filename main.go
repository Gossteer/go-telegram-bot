package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"

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

var numericKeyboard = tgbotapi.NewReplyKeyboard(
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("SHOW"),
	),
)

var commands_map = map[string]func(botSendS *botSendStruct) (tgbotapi.Message, error){
	"help": func(botSendS *botSendStruct) (tgbotapi.Message, error) {
		return botSend(setMessageScruct(botSendS, "Список команд (вводи без скобок, можно в нижнем регистре):\nДля добавления валюты - ADD [название валюты] [сумма (можно десятичное используй точку '.')]\nДля снятия валюты - SUB [название валюты] [сумма (можно десятичное используй точку '.')]\nДля удаления валюты - DEL [название валюты]\nДля демонстрации всех валют в кошельке - SHOW(RUB/USDT)\nПримеры команд: \nADD BTC 0.15\nADD ETH 3.1225\nADD XRP 12.1\nSUB BTC 0.09\nDEL BTC\nSHOW"))
	},
	"start": func(botSendS *botSendStruct) (tgbotapi.Message, error) {
		return botSend(setMessageScruct(botSendS, "Привет, я тестовый бот по криптовалютному кошельку. Все подробности можно узнать через /help"))
	},
}

type CustomCommansMapS struct {
	function     func(botSendS *botSendStruct, commands []string, user_id int) error
	need_command int
}

var custom_commands_map = map[string]CustomCommansMapS{
	"add": {func(botSendS *botSendStruct, commands []string, user_id int) error {
		money, err := strconv.ParseFloat(commands[2], 64)
		if err != nil {
			return err
		}

		if err_set_price := setPrice(user_id, money, commands[1]); err_set_price != nil {
			return errors.New(err_set_price.Error())
		}

		return nil
	}, 3},
	"sub": {func(botSendS *botSendStruct, commands []string, user_id int) error {
		money, err := strconv.ParseFloat(commands[2], 64)
		if err != nil {
			return err
		}
		money = -money

		if err_set_price := setPrice(user_id, -money, commands[1]); err_set_price != nil {
			return errors.New(err_set_price.Error())
		}

		return nil
	}, 3},
	"del": {func(botSendS *botSendStruct, commands []string, user_id int) error {
		delete(db[user_id], commands[1])

		return nil
	}, 2},
	"show": {func(botSendS *botSendStruct, commands []string, user_id int) error {
		return sendShowPrice(botSendS, []string{"RUB", "USDT"}, user_id)
	}, 0},
}

func init() {
	if err := godotenv.Load(); err != nil {
		log.Panic("No .env file found")
	}
}

func main() {
	token, err := getToken()
	if err != nil {
		log.Panic(err)
	}

	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, _ := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil || update.Message.Text == "" {
			continue
		}
		botSendS := botSendStruct{bot, update.Message.Chat.ID, "Я бот Антон"}

		if command, ok := commands_map[strings.ToLower(update.Message.Command())]; ok {
			_, err := command(&botSendS)

			if err != nil {
				botSend(setMessageScruct(&botSendS, err.Error()))
			}

			continue
		}

		commands := strings.Split(update.Message.Text, " ")
		user_id := update.Message.From.ID

		if custom_command, ok := custom_commands_map[strings.ToLower(commands[0])]; ok && chekLenCommand(commands, custom_command.need_command) {
			err := custom_command.function(&botSendS, commands, user_id)

			if err != nil {
				botSend(setMessageScruct(&botSendS, err.Error()))
			}
		} else {
			botSend(setMessageScruct(&botSendS, "Ошибка"))
		}

	}
}

func setPrice(user_id int, money float64, key string) error {
	if _, ok := db[user_id]; !ok {
		db[user_id] = make(wallet)
	}

	if (db[user_id][key] + money) < 0.0 {
		return errors.New("недостаточно средств")
	}

	db[user_id][key] += money

	return nil
}

func sendShowPrice(botSendS *botSendStruct, symbols_to []string, user_id int) error {
	resp := "На счету: \n"
	for key, value := range db[user_id] {

		for _, symbol_to := range symbols_to {
			rub_price, err := getPrice(key, symbol_to)

			if err != nil {
				return err
			}

			resp += fmt.Sprintf("%s (%s): %.2f\n", key, symbol_to, value*rub_price)
		}

		resp += "\n"
	}
	botSend(setMessageScruct(botSendS, resp))

	return nil
}

func getToken() (string, error) {
	token, exists := os.LookupEnv("token")

	if !exists {
		return "", errors.New("токен не обнаружен")
	}

	return token, nil
}

func setMessageScruct(botSendS *botSendStruct, message string) botSendStruct {
	botSendS.message = message

	return *botSendS
}

func botSend(botSendS botSendStruct) (tgbotapi.Message, error) {
	msg := tgbotapi.NewMessage(botSendS.chat_id, botSendS.message)
	msg.ReplyMarkup = numericKeyboard

	return botSendS.bot.Send(msg)
}

func chekLenCommand(command []string, need_len int) bool {
	return (len(command) >= need_len)
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
