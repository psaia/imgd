package main

import (
	"fmt"
	"log"
	"os"

	"github.com/fatih/color"
)

func logLead(s string) string {
	yellow := color.New(color.FgYellow).SprintFunc()
	return fmt.Sprintf("%s%s", yellow("[imgd]: "), s)
}

func prettyLog(f string, v ...interface{}) {
	if _, err := fmt.Println(logLead(prettyLogStr(f, v...))); err != nil {
		log.Fatal(err)
	}
}

func prettyLogStr(f string, v ...interface{}) string {
	blue := color.New(color.FgBlue).SprintFunc()
	return fmt.Sprintf(blue(f)+"\n", v...)
}

func prettyError(f string, v ...interface{}) {
	if _, err := fmt.Printf(logLead(prettyErrorStr(f, v...))); err != nil {
		log.Fatal(err)
	}
}

func prettyErrorStr(f string, v ...interface{}) string {
	red := color.New(color.FgRed).SprintFunc()
	return fmt.Sprintf(red(f)+"\n", v...)
}

func prettyDebug(f string, v ...interface{}) {
	if val, _ := os.LookupEnv("DEBUG"); val == "true" || val == "1" {
		black := color.New(color.FgHiBlack).SprintFunc()
		blue := color.New(color.FgBlue).SprintFunc()
		lead := black("[imgdebug]: ") + blue(f)
		if _, err := fmt.Printf(lead+"\n", v...); err != nil {
			log.Fatal(err)
		}
	}
}
