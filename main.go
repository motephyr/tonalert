package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"sync"
	"time"

	"github.com/motephyr/whalealert/detect"
	"github.com/pyroscope-io/pyroscope/pkg/util/slices"
)

var result struct {
	Ok     bool `json:"ok"`
	Result []struct {
		Type          string `json:"@type"`
		Utime         int    `json:"utime"`
		Data          string `json:"data"`
		TransactionID struct {
			Type string `json:"@type"`
			Lt   string `json:"lt"`
			Hash string `json:"hash"`
		} `json:"transaction_id"`
		Fee        string `json:"fee"`
		StorageFee string `json:"storage_fee"`
		OtherFee   string `json:"other_fee"`
		InMsg      struct {
			Type        string `json:"@type"`
			Source      string `json:"source"`
			Destination string `json:"destination"`
			Value       string `json:"value"`
			FwdFee      string `json:"fwd_fee"`
			IhrFee      string `json:"ihr_fee"`
			CreatedLt   string `json:"created_lt"`
			BodyHash    string `json:"body_hash"`
			MsgData     struct {
				Type      string `json:"@type"`
				Body      string `json:"body"`
				InitState string `json:"init_state"`
			} `json:"msg_data"`
			Message string `json:"message"`
		} `json:"in_msg"`
		OutMsgs []struct {
			Type        string `json:"@type"`
			Source      string `json:"source"`
			Destination string `json:"destination"`
			Value       string `json:"value"`
			FwdFee      string `json:"fwd_fee"`
			IhrFee      string `json:"ihr_fee"`
			CreatedLt   string `json:"created_lt"`
			BodyHash    string `json:"body_hash"`
			MsgData     struct {
				Type string `json:"@type"`
				Text string `json:"text"`
			} `json:"msg_data"`
			Message string `json:"message"`
		} `json:"out_msgs"`
	} `json:"result"`
}

type Config struct {
	duringSecond int64
	recordSecond time.Duration
}

func main() {
	//每5分鐘檢查過去十分鐘的記錄
	config := Config{
		duringSecond: int64(300),
		recordSecond: time.Duration(600),
	}

	fileName2 := "exchange.json"
	byteValue2 := detect.OpenJSONFile(fileName2)
	var exchange map[string]string
	json.Unmarshal(byteValue2, &exchange)

	fileName := "whale.json"
	byteValue := detect.OpenJSONFile(fileName)
	var whale []string
	json.Unmarshal(byteValue, &whale)

	var fileMutex sync.Mutex
	for {
		log.Println("start")
		now := time.Now()
		log.Println("now", now)
		finalResult := config.collectResult(now, exchange)
		notice := []any{}
		for _, x := range finalResult {
			str, _ := x["from"].(string)
			amount, _ := x["amount"].(float64)
			//是鯨魚位址而且大於2百萬的
			if slices.StringContains(whale, str) && amount > 2000000 {
				notice = append(notice, x)
			}
		}

		file, _ := json.MarshalIndent(map[string]any{"finalResult": finalResult, "notice": notice}, "", " ")

		writeResult(*fileMutex, file)
		log.Println("end")

		shouldReturn := pushToGithub()
		log.Println("shouldReturn", shouldReturn)

		later := time.Now()
		log.Println("later", later)
		time.Sleep(time.Duration(config.duringSecond-(later.Unix()-now.Unix())) * time.Second)

	}

}

func writeResult(fileMutex *sync.Mutex, file []byte) {
	fileMutex.Lock()
	err := ioutil.WriteFile("result.json", file, 0644)
	if err != nil {
		log.Println(err)
	}
	fileMutex.Unlock()
}

func pushToGithub() bool {
	cmd := exec.Command("git", "add", ".")
	stdout, err := cmd.Output()

	if err != nil {
		fmt.Println(err.Error())
		return true
	}

	fmt.Println(string(stdout))

	cmd = exec.Command("git", "commit", "-m", "renew")
	stdout, err = cmd.Output()

	if err != nil {
		fmt.Println(err.Error())
		return true
	}

	fmt.Println(string(stdout))

	cmd = exec.Command("git", "push")
	stdout, err = cmd.Output()

	if err != nil {
		fmt.Println(err.Error())
		return true
	}

	fmt.Println(string(stdout))
	return false
}

func (config *Config) collectResult(now time.Time, exchange map[string]string) []map[string]any {
	var wg sync.WaitGroup
	var mux sync.Mutex

	finalResult := []map[string]any{}
	wg.Add(len(exchange))

	for k, v := range exchange {
		go func(k string, v string) {

			defer wg.Done()

			resp, err := http.Get("http://192.168.50.220/api/v2/getTransactions?address=" + k)
			if err != nil {

			}
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)

			json.Unmarshal(body, &result)
			for _, x := range result.Result {
				if x.InMsg.Value != "0" {
					during := time.Duration(now.Unix() - int64(x.Utime))
					amount := detect.GetBalance(x.InMsg.Value, 9)
					if during < config.recordSecond {
						time := time.Unix(int64(x.Utime), 0)
						log.Println("time", time)
						mux.Lock()
						finalResult = append(finalResult, map[string]any{
							"from":   x.InMsg.Source,
							"to":     v,
							"amount": amount,
							"time":   time.String(),
						})
						mux.Unlock()
					}
				}
			}

		}(k, v)

	}

	wg.Wait()
	return finalResult
}

// handle error
// a := []map[string]string{}
// a := []map[string]string{}
// for _, x := range result.Result {
// 	if x.InMsg.Value != "0" {
// 		time := time.Unix(int64(x.Utime), 0)
// 		finalResult = append(finalResult, map[string]string{
// 			"from":   x.InMsg.Source,
// 			"to":     v,
// 			"amount": x.InMsg.Value,
// 			"time":   time.String(),
// 		})
// 	}
// }
// a := [][]map[string]string{}
// for _, x := range result.Result {
// 	b := []map[string]string{}
// 	for _, y := range x.OutMsgs {
// 		b = append(b, map[string]string{
// 			"from":   y.Source,
// 			"to":     y.Destination,
// 			"amount": y.Value,
// 			"time":   strconv.Itoa(x.Utime),
// 		})
// 	}
// 	a = append(a, b)
// 	// a = append(a, map[string]string{
// 	// 	"from":   x.OutMsgs.Source,
// 	// 	"to":     x.OutMsgs.Destination,
// 	// 	"amount": x.OutMsgs.Value,
// 	// 	"time":   strconv.Itoa(x.Utime),
// 	// })
// }
// finalResult = append(finalResult, a)
