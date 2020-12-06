package main

import (
	"fmt"
	"log"
	"os"

	"github.com/fatih/color"
)

func prettyLog(f string, v ...interface{}) {
	if _, err := fmt.Printf(prettyLogStr(f, v)); err != nil {
		log.Fatal(err)
	}
}

func prettyLogStr(f string, v ...interface{}) string {
	yellow := color.New(color.FgYellow).SprintFunc()
	blue := color.New(color.FgBlue).SprintFunc()
	lead := yellow("[imgd]: ") + blue(f)
	if len(v) > 0 {
		return fmt.Sprintf(lead+"\n", v...)
	}
	return fmt.Sprintf(lead + "\n")
}

func prettyError(f string, v ...interface{}) {
	if _, err := fmt.Printf(prettyErrorStr(f, v)); err != nil {
		log.Fatal(err)
	}
}

func prettyErrorStr(f string, v ...interface{}) string {
	yellow := color.New(color.FgYellow).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	lead := yellow("[imgd]: ") + red(f)
	if len(v) > 0 {
		return fmt.Sprintf(lead+"\n", v...)
	}
	return fmt.Sprintf(lead + "\n")
}

func prettyLogMinimal(f string, v ...interface{}) {
	blue := color.New(color.FgBlue).SprintFunc()
	lead := blue(f)
	if _, err := fmt.Printf(lead+"\n", v...); err != nil {
		log.Fatal(err)
	}
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
