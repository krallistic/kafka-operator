package main

import (
	"fmt"
	"os"
	"flag"
)

var (
	version = "0.0.1"

	print bool
)


func init() {
	flag.BoolVar(&print, "print", false, "Show basic information and quit - debug")

	flag.Parse()
}

func Main() int {
	fmt.Println("Started kafka-operator ")
	if print {
		fmt.Println("Operator Version: ", version)
	}

	return 0


}

func main() {
	os.Exit(Main())
}
