package main

import (
	"flag"
	"os"
        "fmt"
        litmus "init-agent/pkg/litmus"
)

var (
        ACTION		    string
	LITMUS_FRONTEND_URL string
	LITMUS_USERNAME     string
	LITMUS_PASSWORD     string
)

func init() {
	flag.StringVar(&ACTION, "action", "", "create|delete litmus agent")
	flag.Parse()

	LITMUS_FRONTEND_URL = os.Getenv("LITMUS_FRONTEND_URL")
	LITMUS_USERNAME = os.Getenv("LITMUS_USERNAME")
	LITMUS_PASSWORD = os.Getenv("LITMUS_PASSWORD")
}

func main() {

	credentials := litmus.Login(LITMUS_FRONTEND_URL, LITMUS_USERNAME, LITMUS_PASSWORD)

	if ACTION == "create" {
		fmt.Println("\nš Start Pre install hook ... š")
		litmus.CreateAgent(credentials)

	} else if ACTION == "delete" {
		fmt.Println("\nš Start Pre delete hook ... š")
		litmus.DeleteAgent(credentials)

	} else {
		fmt.Println("\nā Please choose an action, delete or create")

	}
}
