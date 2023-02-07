package main

import "fmt"

func main() {
	GetVersion(true)
}

var (
	Name    = "gomodules"
	Version = "v1.1.0"
	Date    = "2023-02-07"
)

// GetVersion extract compiled version info
func GetVersion(print bool) (txt string) {
	name := Name
	version := Version
	date := Date
	txt = fmt.Sprintf("%s version %s (%s)", name, version, date)
	if print {
		fmt.Println(txt)
	}
	return
}
