package main

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"time"
)

type Config struct {
	server_host   string
	server_port   string
	max_workers   int
	poll_interval int
	http_timeout  int
}

func NewConfig() Config {
	pserver_host := flag.String("host", "127.0.0.1", "Host to get job from")
	pserver_port := flag.String("port", "8080", "Port of the host")
	pmax_workers := flag.Int("workers", 3, "Maximum number of workers")
	ppoll_interval := flag.Int("pollint", 15, "Poll interval (seconds)")
	phttp_timeout := flag.Int("timeout", 10, "HTTP timeout (seconds)")
	flag.Parse()

	return Config{
		server_host:   *pserver_host,
		server_port:   *pserver_port,
		max_workers:   *pmax_workers,
		poll_interval: *ppoll_interval,
		http_timeout:  *phttp_timeout,
	}
}

func pseudo_uuid() (uuid string) {

	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		fmt.Println("Error: ", err)
		return
	}

	uuid = fmt.Sprintf("%04x-%04x-%04x-%04x-%04x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])

	return
}

func GetAgentId() string {
	return pseudo_uuid()
}

func PostJsonToUrl(url string, jsonData []byte) (StatusCode int, body []byte, err error) {
	request, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Error on NewRequest %v %+v", url, string(jsonData))
		return http.StatusInternalServerError, []byte{}, err
	}
	request.Header.Set("Content-Type", "application/json; charset=UTF-8")
	response, err := myHttpClient.Do(request)
	if err != nil {
		log.Printf("Error on HttpClient.Do %v %+v", url, string(jsonData))
		return http.StatusInternalServerError, []byte{}, err
	}
	defer response.Body.Close()

	body, err = io.ReadAll(response.Body)
	if err != nil {
		log.Printf("Error on reading response.Body %v %+v", url, string(jsonData))
		return response.StatusCode, []byte{}, err
	}
	log.Printf("JSON sent, status: %v url: %v data: %v  body: %v", response.Status, url, string(jsonData), string(body))
	return response.StatusCode, body, nil

}

type AstNode struct {
	Astnode_id int `json:"astnode_id"`
	Task_id    int `json:"task_id"`
	//    parent_astnode_id int
	Operand1       int    `json:"operand1"`
	Operand2       int    `json:"operand2"`
	Operator       string `json:"operator"`
	Operator_delay int    `json:"operator_delay"`
	//    status string
	//    date_ins time.Time
	//    - date_start
	//    - date_done
	//    - agent_id
	Result int64 `json:"result"`
}

func Worker(choper <-chan AstNode, wg *sync.WaitGroup) {
	defer wg.Done()
	url := server_url + "/take-operation-result"

	for operation := range choper {
		func() {
			// щелкаем счетчик свободных в конце - обратно
			defer atomic.AddInt32(&free_workers, 1)
			// вычисляем
			log.Printf("Operation calc start %+v", operation)
			switch {
			case operation.Operator == "+":
				operation.Result = int64(operation.Operand1) + int64(operation.Operand2)
			case operation.Operator == "-":
				operation.Result = int64(operation.Operand1) - int64(operation.Operand2)
			case operation.Operator == "*":
				operation.Result = int64(operation.Operand1) * int64(operation.Operand2)
			case operation.Operator == "/":
				operation.Result = int64(operation.Operand1) / int64(operation.Operand2)
			default:
				log.Printf("Incorrect operator [%v] for operation %+v", operation.Operator, operation)
			}
			// изображаем бурную деятельность
			time.Sleep(time.Duration(operation.Operator_delay) * time.Second)
			// формируем результат
			Json, err := json.Marshal(operation)
			if err != nil {
				log.Printf("Error marshalling to JSON task %+v", operation)
			}
			log.Printf("Operation calc finish %+v", operation)
			// отправляем результат
			if _, _, err := PostJsonToUrl(url, Json); err != nil {
				log.Printf("Request on %v returned error %v", url, err.Error())
			}
		}()
	}
}

func GetOperation(status string) (AstNode, bool) {
	url := server_url + "/give-me-operation"
	//url := "http://ya.ru"
	Json := fmt.Sprintf(`{
		"agent_id": "%v",
		"status": "%v",
		"total_procs": %v,
		"idle_procs": %v
    }`, agent_id, status, config.max_workers, free_workers)
	StatusCode, TaskData, err := PostJsonToUrl(url, []byte(Json))
	if err != nil {
		log.Printf("Request on %v returned error %v", url, err.Error())
		return AstNode{}, false
	}
	if status == "busy" {
		// выполнили "пульс", а операцию не собирались брать, поэтому можем дальше не читать
		return AstNode{}, false
	}
	if StatusCode != http.StatusOK {
		// задания не получили
		return AstNode{}, false
	}
	operation := AstNode{}
	err = json.Unmarshal(TaskData, &operation)
	if err != nil {
		log.Printf("Unmarshal error %v on data %v", err, TaskData)
		return AstNode{}, false
	}
	log.Printf("Got new operation %+v", operation)
	return operation, true
}

func TaskChecker(choper chan<- AstNode, chstop <-chan interface{}) {
	tick := time.NewTicker(time.Duration(config.poll_interval) * time.Second)
	go func() {
		for {
			select {
			case <-tick.C:
				if free_workers > 0 {
					// набираем до упора
					for free_workers > 0 {
						// уменьшаем счетчик свободных
						atomic.AddInt32(&free_workers, -1)
						if oper, ok := GetOperation("ready"); ok {
							// тут отправить операцию воркеру
							choper <- oper
						} else {
							// не дали операцию, освобождаем
							atomic.AddInt32(&free_workers, 1)
							// и выходим из цикла
							break
						}
					}
				} else {
					// нет свободных, просто сделаем "пульс" и скажем серверу что заняты
					GetOperation("busy")
				}
			case <-chstop:
				tick.Stop()
				close(choper)
				return
			}
		}

	}()

}

var config = NewConfig()
var agent_id = GetAgentId()
var free_workers = int32(config.max_workers)
var server_url = "http://" + config.server_host + ":" + config.server_port
var myHttpClient = &http.Client{Timeout: time.Duration(config.http_timeout) * time.Second}

func main() {

	log.Printf("Agent started with agent_id=%v", agent_id)
	log.Printf("Config %+v", config)

	choper := make(chan AstNode)
	chstop := make(chan interface{})
	wg := new(sync.WaitGroup)

	// Создаем воркеров
	for i := 0; i < config.max_workers; i++ {
		wg.Add(1)
		go Worker(choper, wg)
	}

	// поллер заданий, он же heartbeat
	TaskChecker(choper, chstop)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	<-c
	log.Print("Exit on ctrl-c signal")

	log.Printf("Waiting for workers running workers to finish. Busy workers %v of total %v", config.max_workers-int(free_workers), config.max_workers)
	close(chstop)
	wg.Wait()
	log.Print("All workers finished")

}

/*
func Heartbeat() {
	tick := time.NewTicker(time.Duration(config.poll_interval) * time.Second)
	url := server_url + "/give-me-astnode"
	//url := "http://ya.ru"
	go func() {
		for {
			<-tick.C
			Json := fmt.Sprintf(`{
				"agent_id": "%v",
				"total_procs": "%v",
				"idle_procs": "%v"
			}`,agent_id, config.max_workers, free_workers)
			if _, _, err := PostJsonToUrl(url,[]byte(Json)); err != nil {
				log.Printf("Request on %v returned error %v", url, err.Error())
			}

		}

	}()
}
func Heartbeat() {
	tick := time.NewTicker(time.Duration(config.poll_interval) * time.Second)
	url := server_url + "/heartbeat"
	//url := "http://ya.ru"
	go func() {
		for {
			<-tick.C
			Json := fmt.Sprintf(`{
				"agent_id": "%v",
				"total_procs": "%v",
				"idle_procs": "%v"
			}`,agent_id, config.max_workers, free_workers)
			if _, _, err := PostJsonToUrl(url,[]byte(Json)); err != nil {
				log.Printf("Request on %v returned error %v", url, err.Error())
			}

		}

	}()
}
func GetJsonFromUrl(url string, target interface{}) error {
    r, err := myHttpClient.Get(url)
    if err != nil {
        return err
    }
    defer r.Body.Close()

    return json.NewDecoder(r.Body).Decode(target)
}

func SendHTTPRequest(url string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := myHttpClient.Do(req)
	if err != nil {
		return "", err
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

*/
