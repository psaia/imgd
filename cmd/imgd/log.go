package main

import (
	"fmt"
	"log"
	"os"
	"runtime"

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
		PrintMemUsage()
	}
}

// PrintMemUsage outputs the current, total and OS memory being used. As well as the number
// of garage collection cycles completed.
func PrintMemUsage() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	// For info on each, see: https://golang.org/pkg/runtime/#MemStats
	fmt.Printf("Alloc = %v MiB", bToMb(m.Alloc))
	fmt.Printf("\tTotalAlloc = %v MiB", bToMb(m.TotalAlloc))
	fmt.Printf("\tSys = %v MiB", bToMb(m.Sys))
	fmt.Printf("\tNumGC = %v\n", m.NumGC)
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}
