package main

import (
	"os"

	"github.com/taylormonacelli/farwrite"
)

func main() {
	code := farwrite.Execute()
	os.Exit(code)
}
