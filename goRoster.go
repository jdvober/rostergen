package main

import (
	auth "github.com/jdvober/goGoogleAuth"
)

func main() {
	client := auth.Authorize()
	spreadsheetId := "1HRfK4yZERLWd-OcDZ8pJRirdzdkHln3SUtIfyGZEjNk"
	rangeData := "Master!A2"
}
