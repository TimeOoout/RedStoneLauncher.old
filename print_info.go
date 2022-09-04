package main

import (
	"fmt"
	"github.com/fatih/color"
)

func Result(format string, args ...interface{}) {
	fmt.Printf(color.CyanString("[Result] \n"+format+"\n", args...))
}

func Info(format string, args ...interface{}) {
	fmt.Printf(color.BlueString("[INFO] "+format+"\n", args...))
}

func Warn(format string, args ...interface{}) {
	fmt.Printf(color.YellowString("[Warning] "+format+"\n", args...))
}

func Error(format string, args ...interface{}) {
	fmt.Printf(color.RedString("[Error] "+format+"\n", args...))
}

func Debug(format string, args ...interface{}) {
	fmt.Printf(color.GreenString("[Debug] "+format+"\n", args...))
}

func Simple(format string, args ...interface{}) {
	fmt.Printf(color.WhiteString(format+"\n", args...))
}
