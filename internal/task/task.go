package task

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strconv"
	"sync"
	"time"

	"github.com/Mur466/distribcalc/internal/cfg"
	l "github.com/Mur466/distribcalc/internal/logger"
	"go.uber.org/zap"
)

type Node struct {
	Node_id        int `json:"node_id"`
	Task_id        int
	Parent_node_id int
	Child1_node_id int
	Child2_node_id int
	Operand1       int    `json:"operand1"`
	Operand2       int    `json:"operand2"`
	Operator       string `json:"operator"`
	Operator_delay int    `json:"operator_delay"`
	Status         string `json:"status"` // (parsing, "error", waiting - ждем результатов других выражений, ready - оба операнда вычислены, in progress - передано в расчет, done - есть результат)
	Message        string `json:"message"`
	Date_ins       time.Time
	Date_start     time.Time
	Date_done      time.Time
	Agent_id       string `json:"agent_id"`
	Result         int64  `json:"result"`
}

type NodeStatusInfo struct {
	Agent_id string
	Result   int64
	Message  string
}

type Task struct {
	Task_id      int
	Ext_id       string // внешний идентификатор для идемпотентности
	Expr         string
	Result       int64
	Status       string  // (parsing, error, ready, in progress, done)
	Message      string  // текстовое сообщение с результатом/ошибкой
	TreeSlice    []*Node // Дерево Abstract Syntax Tree
	Mx           sync.Mutex
	DateCreated  time.Time
	DateFinished time.Time
}

var Tasks map[int]*Task = make(map[int]*Task, 0)

func NewTask(expr string, ext_id string) *Task {
	double := CheckUnique(ext_id)
	if double != nil {
		return double
	}

	t := &Task{Task_id: -1, Expr: expr, Ext_id: ext_id, TreeSlice: make([]*Node, 0), DateCreated: time.Now()}
	t.SetStatus("parsing", TaskStatusInfo{})

	root := &Node{Task_id: t.Task_id}
	t.AddNode(-1, root)

	parsedtree, err := parser.ParseExpr(expr)
	if err != nil {
		t.SetStatus("error", TaskStatusInfo{Message: err.Error()})
		return t
	}

	err = t.buildtree(parsedtree, t.TreeSlice[0])
	if err != nil {
		t.SetStatus("error", TaskStatusInfo{Message: err.Error()})
		return t
	}
	//t.SaveTask()
	t.SetStatus("ready", TaskStatusInfo{})
	return t
}

// Проверяем, что задание с таким ext_id уже есть
// Если находим, вернем его
func CheckUnique(Ext_id string) *Task {
	if Ext_id == "" {
		// если ключ не указали, не проверяем
		return nil
	}

	for _, t := range Tasks {
		if t.Ext_id == Ext_id {
			return t
		}
	}
	return nil
}

func (t *Task) buildtree(parsedtree ast.Expr, parent *Node) error {

	switch n := parsedtree.(type) {
	case *ast.BasicLit:
		//сюда попасть не должны
		l.Logger.Error("Unexpected switch case",
			zap.String("n.type", "*ast.BasicLit"))
	case *ast.UnaryExpr:
		switch n.Op {
		case token.ADD:
			parent.Operator = "+"
		case token.SUB:
			parent.Operator = "-"
		default:
			return unsup(reflect.TypeOf(parsedtree))
		}
		parent.Operand1 = 0
		parent.Child1_node_id = -1
		parent.Operator_delay = GetOperatorDelay(parent.Operator)

		// тут хитро - в дереве унарный оператор с выражением X,
		// но мы присваиваем его ВТОРОЙ ноде, чтобы -1 превратить в 0 - 1
		switch x := n.X.(type) {
		case *ast.BasicLit:
			// вычислять не нужно
			if x.Kind != token.INT {
				return unsup(x.Kind)
			}
			parent.Operand2, _ = strconv.Atoi(x.Value)
			parent.Status = "ready"
		default:
			parent.Status = "waiting" // придется вычислять операнд
			childX := t.AddNode(parent.Node_id, &Node{})
			errX := t.buildtree(n.X, childX)
			parent.Child2_node_id = childX.Node_id
			if errX != nil {
				return errX
			}
		}
		return nil
	case *ast.BinaryExpr:
		//var operator string
		switch n.Op {
		case token.ADD:
			parent.Operator = "+"
		case token.SUB:
			parent.Operator = "-"
		case token.MUL:
			parent.Operator = "*"
		case token.QUO:
			parent.Operator = "/"
		default:
			return unsup(n.Op)
		}
		parent.Operator_delay = GetOperatorDelay(parent.Operator)
		parent.Status = "ready" // оптимистично считаем, что оба операнда будут на блюдечке

		switch x := n.X.(type) {
		case *ast.BasicLit:
			// вычислять не нужно
			if x.Kind != token.INT {
				return unsup(x.Kind)
			}
			parent.Operand1, _ = strconv.Atoi(x.Value)
		default:
			parent.Status = "waiting" // придется вычислять операнд
			childX := t.AddNode(parent.Node_id, &Node{})
			errX := t.buildtree(n.X, childX)
			parent.Child1_node_id = childX.Node_id
			if errX != nil {
				return errX
			}
		}

		switch y := n.Y.(type) {
		case *ast.BasicLit:
			// вычислять не нужно
			if y.Kind != token.INT {
				return unsup(y.Kind)
			}
			parent.Operand2, _ = strconv.Atoi(y.Value)
		default:
			parent.Status = "waiting" // придется вычислять операнд
			childY := t.AddNode(parent.Node_id, &Node{})
			errY := t.buildtree(n.Y, childY)
			parent.Child2_node_id = childY.Node_id
			if errY != nil {
				return errY
			}
		}
		return nil
	case *ast.ParenExpr:
		return t.buildtree(n.X, parent)
	}
	return unsup(reflect.TypeOf(parsedtree))
}

func unsup(i interface{}) error {

	return fmt.Errorf("%v unsupported", i)
}

/*
func NewNode(operand1, operand2 int, operator, status string) *Node {
	return &Node{
		Node_id:        -1,
		Operand1:       operand1,
		Operand2:       operand2,
		Operator:       operator,
		Status:         status,
		Child1_node_id: -1,
		Child2_node_id: -1,
	}
}
*/
func (t *Task) AddNode(parent_id int, node *Node) *Node {
	node.Node_id = len(t.TreeSlice)
	node.Date_ins = time.Now()
	node.Parent_node_id = parent_id
	node.Task_id = t.Task_id
	//node.Tree = t
	node.Child1_node_id = -1
	node.Child2_node_id = -1
	t.TreeSlice = append(t.TreeSlice, node)
	return node

}

type TaskStatusInfo struct {
	Result  int64
	Message string
}

func (t *Task) SetStatus(status string, info TaskStatusInfo) {
	// проверим, что кто-то другой не изменил уже наш статус до нас
	if status == t.Status {
		// делать нечего
		return
	}
	switch status {
	default:
		l.Logger.Error("Invalid status",
			zap.Int("task_id", t.Task_id),
			zap.String("status", status),
		)
		return
	case "parsing", "ready", "in progress":
		l.Logger.Info("Task status changed",
			zap.Int("task_id", t.Task_id),
			zap.String("status", status),
		)
		t.Status = status
	case "done":
		t.Result = info.Result
		t.Message = fmt.Sprintf("Calculation complete. Result = %v", t.Result)
		t.DateFinished = time.Now()
		t.Status = "done"
		l.Logger.Info("Task status complete",
			zap.Int("task_id", t.Task_id),
			zap.Int64("result", t.Result),
		)
	case "error":
		t.Message = fmt.Sprintf("Calculation failed. Error = %v", info.Message)
		t.DateFinished = time.Now()
		t.Status = "error"
		l.Logger.Error("Task failed",
			zap.Int("task_id", t.Task_id),
			zap.String("message", info.Message),
		)
	}

	t.SaveTask()
}

// выбираем ожидающую операцию и переводим ее в процесс
func (t *Task) GetWaitingNodeAndSetProcess(agent_id string) (*Node, bool) {
	for _, n := range t.TreeSlice {
		ret := false
		func() {
			t.Mx.Lock()
			defer t.Mx.Unlock()
			// тут мы 100% одни
			if n.Status == "ready" {
				n.SetToProcess(agent_id)
				ret = true
			}
		}()
		if ret {
			return n, true
		}
	}
	// нет операций готовых к вычислению
	return nil, false
}

func (t *Task) SetNodeStatus(node_id int, status string, info NodeStatusInfo) {
	if node_id > len(t.TreeSlice)-1 || node_id < 0 {
		l.Logger.Error("Node id out of bounds",
			zap.Int("task_id", t.Task_id),
			zap.Int("task_id", node_id),
		)
		return
	}
	n := t.TreeSlice[node_id]
	switch status {
	default:
		l.Logger.Error("Invalid status",
			zap.Int("task_id", t.Task_id),
			zap.Int("node_id", node_id),
			zap.String("status", status),
		)
	case "in progress": // передано в расчет
		n.Agent_id = info.Agent_id
		n.Date_ins = time.Now()
	case "done", "error": // есть результат или ошибка
		n.Date_done = time.Now()
		n.Result = int64(info.Result)
		n.Message = info.Message
	case "parsing",
		"waiting", //ждем результатов других выражений
		"ready":   // готов к вычислению

	}
	n.Status = status
	l.Logger.Info("Node new status",
		zap.Int("task_id", t.Task_id),
		zap.Int("node_id", node_id),
		zap.String("status", status),
	)

	t.SaveTask()

	// доп. обработка после сохранения в БД
	if t.Status != "in progress" {
		// делать нечего, можно забыть про результат
		return
	}

	if n.Status == "done" {
		parent_id := n.Parent_node_id
		if parent_id == -1 {
			// если посчитали корневой узел, то значит выражение тоже
			t.SetStatus("done", TaskStatusInfo{Result: info.Result})
		} else {
			parent := t.TreeSlice[parent_id]
			// Запишем результат в родителя
			if parent.Child1_node_id == node_id {
				// мы - первая дочка
				parent.Operand1 = int(info.Result)
			} else {
				parent.Operand2 = int(info.Result)
			}
			// проверим, может и родителя можно считать?
			if parent.Status == "waiting" &&
				(parent.Child1_node_id == -1 || // нет дочки
					t.TreeSlice[parent.Child1_node_id].Status == "done") &&
				(parent.Child2_node_id == -1 || // нет дочки
					t.TreeSlice[parent.Child2_node_id].Status == "done") {
				// дочек нет или они посчитаны, можем считать папу
				t.SetNodeStatus(parent_id, "ready", NodeStatusInfo{})
			}
		}
	} else if n.Status == "error" {
		// ошибка в операци, отменяем задание и все ожидающие ноды
		t.SetStatus("error", TaskStatusInfo{Message: info.Message})
		for _, n := range t.TreeSlice {
			if n.Status == "waiting" || n.Status == "ready" {
				t.SetNodeStatus(n.Node_id, "error", NodeStatusInfo{Message: "Some other node has error"})
			}
		}
	}

}

// Получить задержку по оператору из конфига
func GetOperatorDelay(operator string) int {
	switch operator {
	case "+":
		return cfg.Cfg.DelayForAdd
	case "-":
		return cfg.Cfg.DelayForSub
	case "*":
		return cfg.Cfg.DelayForMul
	case "/":
		return cfg.Cfg.DelayForDiv
	default:
		return 0 // не бывает
	}

}

func (t *Task) DateCreatedFmt() string {
	if t.DateCreated.IsZero() {
		return "N/A"
	}
	return t.DateCreated.Format("2006/01/02 15:04:05")
}

func (t *Task) DateFinishedFmt() string {
	if t.DateFinished.IsZero() {
		return "N/A"
	}
	return t.DateFinished.Format("2006/01/02 15:04:05")
}
