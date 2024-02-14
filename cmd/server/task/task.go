package task

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"reflect"
	"strconv"
	"sync"
	"time"
)

type AstNode struct {
	Astnode_id        int `json:"astnode_id"`
	Task_id           int
	Parent_astnode_id int
	Child1_astnode_id int
	Child2_astnode_id int
	Tree              *Task
	Operand1          int    `json:"operand1"`
	Operand2          int    `json:"operand2"`
	Operator          string `json:"operator"`
	Operator_delay    int    `json:"operator_delay"`
	Status            string // (parsing, failed, waiting - ждем результатов других выражений, ready - оба операнда вычислены, in progress - передано в расчет, done - есть результат)
	Date_ins          time.Time
	Date_start        time.Time
	Date_done         time.Time
	Agent_id          string
	Result            int64 `json:"result"`
}

type NodeStatusInfo struct {
	Agent_id string
	Result int64
}

type Task struct {
	Task_id   int
	Ext_id    string // внешний идентификатор для идемпотентности
	Expr      string
	Result    int64
	Status    string     // (parsing, error, ready, in progress, done)
	Message   string     // текстовое сообщение с результатом/ошибкой
	TreeSlice []*AstNode // Дерево Abstract Syntax Tree
	mx		sync.Mutex
}

var task_count int
func NewTask(expr string, ext_id string) *Task {
	task_count++
	t := &Task{Task_id: task_count, Expr: expr, Ext_id: ext_id, TreeSlice: make([]*AstNode, 0)}
	t.SetStatus("parsing",TaskStatusInfo{})
	root := &AstNode{Task_id: t.Task_id}
	t.Add(-1, root)

	parsedtree, err := parser.ParseExpr(expr)
	if err != nil {
		//t.SetFailed(err.Error())
		t.SetStatus("error", TaskStatusInfo{Message: err.Error()})
		return t
	}

	err = t.buildtree(parsedtree, t.TreeSlice[0])
	if err != nil {
		//t.SetFailed(err.Error())
		t.SetStatus("error", TaskStatusInfo{Message: err.Error()})
		return t
	}
	t.SetStatus("ready",TaskStatusInfo{})
	return t
}

func (t *Task) buildtree(parsedtree ast.Expr, parent *AstNode) error {

	switch n := parsedtree.(type) {
	/*
		сюда попасть не должны
		case *ast.BasicLit:
			if n.Kind != token.INT {
				return unsup2(n.Kind)
			}
			i, _ := strconv.Atoi(n.Value)
			return i, nil
	*/
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
		parent.Status = "ready" // оптимистично считаем, что оба операнда будут на блечке

		switch x := n.X.(type) {
		case *ast.BasicLit:
			// вычислять не нужно
			if x.Kind != token.INT {
				return unsup(x.Kind)
			}
			parent.Operand1, _ = strconv.Atoi(x.Value)
		default:
			parent.Status = "waiting" // придется вычислять операнд
			childX := t.Add(parent.Astnode_id, &AstNode{})
			errX := t.buildtree(n.X, childX)
			parent.Child1_astnode_id = childX.Astnode_id
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
			childY := t.Add(parent.Astnode_id, &AstNode{})
			errY := t.buildtree(n.Y, childY)
			parent.Child2_astnode_id = childY.Astnode_id
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
/*
func evalX (X ast.Expr, operand *int) {

}
*/
func unsup(i interface{}) error {

	return fmt.Errorf("%v unsupported", i)
}

func NewAstNode(operand1, operand2 int, operator, status string) *AstNode {
	return &AstNode{
		Operand1: operand1,
		Operand2: operand2,
		Operator: operator,
		//	Operator_delay    int
		Status: status,
		Child1_astnode_id: -1,
		Child2_astnode_id: -1,
		
	}
}
func (t *Task) Add(parent_id int, node *AstNode) *AstNode {
	node.Astnode_id = len(t.TreeSlice)
	node.Date_ins = time.Now()
	node.Parent_astnode_id = parent_id
	node.Task_id = t.Task_id
	node.Tree = t
	node.Child1_astnode_id = -1
	node.Child2_astnode_id = -1
	t.TreeSlice = append(t.TreeSlice, node)
	return node

}
type TaskStatusInfo struct {
	Result int64
	Message string
}

func (t *Task) SetStatus(status string, info TaskStatusInfo) {
	switch status{
	default:
		log.Printf("Invalid status. TaskId: %v, Status:%v", t.Task_id, status)
		return
	case "parsing", "ready", "in progress":
		 log.Printf("New task status. TaskId: %v, Status:%v", t.Task_id, status)
		 t.Status = status
	case "done":
		t.Result = info.Result
		log.Printf("Task complete. TaskId: %v, result: %v", t.Task_id, t.Result)
		t.Message = fmt.Sprintf("Calculation complete. Result = %v", t.Result)
		t.Status = "done"
	case "error":
		log.Printf("Task failed. TaskId: %v, error: %v", t.Task_id, info.Message)
		t.Message = fmt.Sprintf("Calculation failed. Error = %v", info.Message)
		t.Status = "failed"
	}

	//todo db
}
/*
func (t *Task) SetResult(result int64) {
	t.Result = result
	log.Printf("Task complete. TaskId: %v, result: %v", t.Task_id, result)
	t.Message = fmt.Sprintf("Calculation complete. Result = %v", result)
	t.Status = "done"
	// todo db
}

func (t *Task) SetFailed(message string) {
	log.Printf("Task failed. TaskId: %v, error: %v", t.Task_id, message)
	t.Message = fmt.Sprintf("Calculation failed. Error = %v", message)
	t.Status = "failed"
	// todo db
}
*/

// атомарно выбираем ожидающую операцию и переводим ее в процесс
func (t *Task) GetWaitingNodeAndSetProcess(agent_id string) (*AstNode, bool) {
	for _, n := range t.TreeSlice {
		if n.Status == "ready" {
			t.SetNodeStatus(n.Astnode_id,"in progress", NodeStatusInfo{Agent_id: agent_id})
			return n, true
		}
	}
	// нет операций готовых к вычислению
	return nil, false
}


func (t *Task) SetNodeStatus(AstNodeId int, status string, info NodeStatusInfo) {
	if AstNodeId > len(t.TreeSlice)-1 || AstNodeId < 0 {
		log.Printf("Node id out of bounds. TaskId: %v, NodeId: %v", t.Task_id, AstNodeId)
		return
	}
	switch status{
	default:
		log.Printf("Invalid status. TaskId: %v, NodeId: %v, Status:%v", t.Task_id, AstNodeId, status)
	case "in progress":  // передано в расчет
		t.TreeSlice[AstNodeId].Agent_id=info.Agent_id
	case "done": // есть результат
		t.TreeSlice[AstNodeId].Result=int64(info.Result)
	case "parsing", "failed", 
		 "waiting" /*ждем результатов других выражений*/,
		 "ready" /* оба операнда вычислены*/:

	}
	t.TreeSlice[AstNodeId].Status = status
	log.Printf("Node new status. TaskId: %v, NodeId: %v, Status:%v, Info: %v", t.Task_id, AstNodeId, status, info)
	// todo db

	// доп. обработка после сохранения в БД
	if status == "done" {		
		parent_id := t.TreeSlice[AstNodeId].Parent_astnode_id
		if parent_id == -1 {
			// если посчитали корневой узел, то значит выражение тоже
			//t.SetResult(info.Result)
			t.SetStatus("done", TaskStatusInfo{Result:info.Result})
		} else {
			t.mx.Lock()
			defer t.mx.Unlock()
			parent := t.TreeSlice[parent_id]
			// Запишем результат в родителя
			if parent.Child1_astnode_id == AstNodeId {
				// мы - первая дочка
				parent.Operand1 = int(info.Result)
			} else {
				parent.Operand2 = int(info.Result)
			}
			// проверим, может и родителя можно считать?
			if parent.Status == "waiting" &&
				(parent.Child1_astnode_id == -1 || // нет дочки
			    t.TreeSlice[parent.Child1_astnode_id].Status=="done") &&
			    (parent.Child2_astnode_id == -1 || // нет дочки
				t.TreeSlice[parent.Child2_astnode_id].Status=="done") {
				// дочек нет или они посчитаны, можем считать папу
				t.SetNodeStatus(parent_id, "ready", NodeStatusInfo{})
			}
		}
	
	}


}

// Получить задержку по оператору из конфига
func GetOperatorDelay(operator string) int {
	switch operator {
	case "+":
		return 10
	case "-":
		return 20
	case "*":
		return 30
	case "/":
		return 40
	default:
		return 0 // не бывает
	}

}
