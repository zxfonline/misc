package main

import (
	"fmt"
	"strings"

	"topcrown.com/centerserver/misc/csvconfig"
	"topcrown.com/centerserver/misc/keyword"
)

func main() {
	csvconfig.Init("./example/csv", ".csv")
	err1 := csvconfig.Load([]string{"badwords"})
	if err1 != nil {
		panic(err1)
	}
	recs := csvconfig.GetAll("badwords")
	words := make([]string, 0, len(recs))
	for _, rec := range recs {
		fields := rec.Fields
		word := fields["word"]
		words = append(words, word)
	}
	kword := keyword.NewKeyWord()
	kword.Init(words)
	keyword.SetGlobal(kword)
	bol := haveBadWords("FUCK")
	fmt.Println(bol)
}

func haveBadWords(str string) bool {
	lower := strings.ToLower(str)
	return keyword.Search(lower)
}
