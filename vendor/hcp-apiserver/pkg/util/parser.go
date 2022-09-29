package util

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

func Parser(req *http.Request, input interface{}) {
	jsonDataFromHttp, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(jsonDataFromHttp))
	err = json.Unmarshal(jsonDataFromHttp, input)
	defer req.Body.Close()
	if err != nil {
		fmt.Println(err.Error())
	}
}
