package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"time"

	"github.com/JohnCGriffin/overflow"

	"github.com/Mur466/distribcalc/internal/agent"
	"github.com/Mur466/distribcalc/internal/task"
	"github.com/Mur466/distribcalc/internal/utils"
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

func GetAgentId() string {
	return utils.Pseudo_uuid()
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

func Worker(choper <-chan task.Node, wg *sync.WaitGroup) {
	defer wg.Done()
	url := server_url + "/take-operation-result"

	for operation := range choper {
		func() {
			// щелкаем счетчик свободных в конце - обратно
			defer atomic.AddInt32(&free_workers, 1)
			// вычисляем
			log.Printf("Operation calc start %+v", operation)
			var no_overfl bool = true
			switch {
			case operation.Operator == "+":
				//operation.Result = int64(operation.Operand1) + int64(operation.Operand2)
				operation.Result, no_overfl = overflow.Add64(int64(operation.Operand1), int64(operation.Operand2))
			case operation.Operator == "-":
				//operation.Result = int64(operation.Operand1) - int64(operation.Operand2)
				operation.Result, no_overfl = overflow.Sub64(int64(operation.Operand1), int64(operation.Operand2))
			case operation.Operator == "*":
				//operation.Result = int64(operation.Operand1) * int64(operation.Operand2)
				operation.Result, no_overfl = overflow.Mul64(int64(operation.Operand1), int64(operation.Operand2))
			case operation.Operator == "/":
				if operation.Operand2 == 0 {
					operation.Status = "error"
					operation.Message = "Division by zero"
				} else {
					operation.Result = int64(operation.Operand1) / int64(operation.Operand2)
					operation.Result, no_overfl = overflow.Div64(int64(operation.Operand1), int64(operation.Operand2))
				}
			default:
				operation.Status = "error"
				operation.Message = "Incorrect operator [" + operation.Operator + "]"
				log.Printf("Incorrect operator [%v] for operation %+v", operation.Operator, operation)
			}
			if !no_overfl {
				operation.Status = "error"
				operation.Message = "Overflow"
			}
			if operation.Status != "error" {
				// изображаем бурную деятельность
				time.Sleep(time.Duration(operation.Operator_delay) * time.Second)
				operation.Status = "done"
			}
			operation.Agent_id = agent_id
			// формируем результат
			Json, err := json.Marshal(operation)
			if err != nil {
				log.Printf("Error marshalling to JSON task %+v", operation)
			}
			log.Printf("Operation calc finish %+v status", operation)
			// отправляем результат
			if _, _, err := PostJsonToUrl(url, Json); err != nil {
				log.Printf("Request on %v returned error %v", url, err.Error())
			}
		}()
	}
}

func GetOperation(status string) (task.Node, bool) {
	url := server_url + "/give-me-operation"
	Json, err := json.Marshal(agent.Agent{
		AgentId:    agent_id,
		Status:     status,
		TotalProcs: config.max_workers,
		IdleProcs:  int(free_workers),
	})
	if err != nil {
		log.Printf("Error marshalling to JSON task agent status")
		return task.Node{}, false
	}

	StatusCode, TaskData, err := PostJsonToUrl(url, Json)
	if err != nil {
		log.Printf("Request on %v returned error %v", url, err.Error())
		return task.Node{}, false
	}
	if status == "busy" {
		// выполнили "пульс", а операцию не собирались брать, поэтому можем дальше не читать
		return task.Node{}, false
	}
	if StatusCode != http.StatusOK {
		// задания не получили
		return task.Node{}, false
	}
	operation := task.Node{}
	err = json.Unmarshal(TaskData, &operation)
	if err != nil {
		log.Printf("Unmarshal error %v on data %v", err, TaskData)
		return task.Node{}, false
	}
	log.Printf("Got new operation %+v", operation)
	return operation, true
}

func DinDon(choper chan<- task.Node) {
	if free_workers > 0 {
		// набираем до упора
		for free_workers > 0 {
			if oper, ok := GetOperation("ready"); ok {
				// уменьшаем счетчик свободных
				atomic.AddInt32(&free_workers, -1)
				// тут отправить операцию воркеру
				choper <- oper
			} else {
				// не дали операцию, выходим из цикла
				break
			}
		}
	} else {
		// нет свободных, просто сделаем "пульс" и скажем серверу что заняты
		GetOperation("busy")
	}
}

func TaskChecker(choper chan<- task.Node, chstop <-chan interface{}) {
	tick := time.NewTicker(time.Duration(config.poll_interval) * time.Second)
	go func() {
		for {
			select {
			case <-tick.C:
				// таймер прозвенел
				DinDon(choper)
			case <-chstop:
				tick.Stop()
				close(choper)
				return
			}
		}

	}()
	// первый раз не ждем таймера
	DinDon(choper)
}

var config = NewConfig()
var agent_id = GetAgentId()
var free_workers = int32(config.max_workers)
var server_url = "http://" + config.server_host + ":" + config.server_port
var myHttpClient = &http.Client{Timeout: time.Duration(config.http_timeout) * time.Second}

func main() {

	log.Printf("Agent started with agent_id=%v", agent_id)
	log.Printf("Config %+v", config)

	choper := make(chan task.Node)
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
