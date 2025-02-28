package main

import (
	"context"
	"encoding/json"
	"fmt"
	"goBangumiAPI/bangumiAPI"
)

var accessToken = "JuTyz4XYdeO16FU8mXdUGDIMjfx5JeZif2OKIoHE"

func decodeData(data any) (map[string]interface{}, error) {
	jsonData, jsonErr := json.MarshalIndent(data, "", "\t")
	if jsonErr != nil {
		return nil, jsonErr
	}
	var outData map[string]interface{}
	jsonErr2 := json.Unmarshal(jsonData, &outData)
	if jsonErr2 != nil {
		return nil, jsonErr2
	}
	return outData, nil
}

func main() {
	ctx := context.Background()
	cli := bangumiAPI.NewClient("test",
		"https://api.bgm.tv",
		"")

	//resp, err := cli.SearchMediumSubjectByKeywords(ctx, accessToken, "CLANNAD", 0, "", 0, 25)
	resp, err := cli.GetSubject(ctx, accessToken, "2388", nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	jsonData, jsonErr := json.MarshalIndent(resp, "", "\t")
	if jsonErr != nil {
		fmt.Println(jsonData)
		return
	}
	fmt.Printf("%+v", string(jsonData))
}
