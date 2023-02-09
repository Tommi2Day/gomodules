package main

import "fmt"

func main() {
	GetVersion(true)
}

var (
	Name    = "golibs"
	Version = "v1.3.0"
	Date    = "2023-02-09"
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
