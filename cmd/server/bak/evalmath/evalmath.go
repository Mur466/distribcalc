package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strconv"

	"github.com/Mur466/distribcalc/cmd/server/task"
)

var tests = []string{

	"1+(3*2)/5", // my example
	"(1+3)*7",   // 28, example from task description.
	"1+3*7",     // 22, shows operator precedence.
	"7",         // 7, a single literal is a valid expression.
	"7/3",       // eval only does integer math.
	"7.3",       // this parses, but we disallow it in eval.
	"7^3",       // parses, but disallowed in eval.
	"go",        // a valid keyword, not valid in an expression.
	"3@7",       // error message is "illegal character."
	"",          // EOF seems a reasonable error message.
}

func main() {

	for _, exp := range tests {
		if r, err := parseandbuild(exp, 0); err == nil {
			fmt.Println(r.Expr)
			for _, v := range r.TreeSlice {
				fmt.Printf("%v, %v: %v %v %v = %v \n", v.Astnode_id, v.Status, v.Operand1, v.Operator, v.Operand2, v.Result)
			}
			fmt.Println()
		} else {
			fmt.Printf("%s: %v\n", exp, err)
		}
	}
}

func parseandbuild(expr string, task_id int) (*task.Task, error) {

	t := task.NewTask(expr, task_id)
	parsedtree, err := parser.ParseExpr(expr)
	if err != nil {
		return &task.Task{}, err
	}

	err = buildtree(parsedtree, t.TreeSlice[0])
	if err != nil {
		return &task.Task{}, err
	}
	return t, nil

}

func buildtree(parsedtree ast.Expr, parent *task.AstNode) error {

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
		parent.Operator_delay = task.GetOperatorDelay(parent.Operator)
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
			childX := task.AstNode{
				// кажется тут вообще ничего не знаем?
			}
			parent.Tree.Add(parent.Astnode_id, &childX)
			errX := buildtree(n.X, &childX)
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
			childY := parent.Tree.Add(parent.Astnode_id, &task.AstNode{
				// кажется тут вообще ничего не знаем?
			})
			//errY := buildtree(n.Y, &(parent.Tree.Tree[len(parent.Tree.Tree)-1]))
			errY := buildtree(n.Y, childY)
			if errY != nil {
				return errY
			}
		}
		return nil
	case *ast.ParenExpr:
		return buildtree(n.X, parent)
	}
	return unsup(reflect.TypeOf(parsedtree))
}

func unsup(i interface{}) error {

	return fmt.Errorf("%v unsupported", i)
}

/*
func parseAndEval(exp string) (int, error) {

	tree, err := parser.ParseExpr(exp)
	if err != nil {
		return 0, err
	}
	return eval(tree)
}

func eval(tree ast.Expr) (int, error) {

	switch n := tree.(type) {
	case *ast.BasicLit:
		if n.Kind != token.INT {
			return unsup(n.Kind)
		}
		i, _ := strconv.Atoi(n.Value)
		return i, nil
	case *ast.BinaryExpr:
		switch n.Op {
		case token.ADD, token.SUB, token.MUL, token.QUO:
		default:
			return unsup(n.Op)
		}
		x, err := eval(n.X)
		if err != nil {
			return 0, err
		}
		y, err := eval(n.Y)
		if err != nil {
			return 0, err
		}
		switch n.Op {
		case token.ADD:
			return x + y, nil
		case token.SUB:
			return x - y, nil
		case token.MUL:
			return x * y, nil
		case token.QUO:
			return x / y, nil
		}
	case *ast.ParenExpr:
		return eval(n.X)
	}
	return unsup(reflect.TypeOf(tree))
}
*/

/*
func unsup(i interface{}) (int, error) {

	return 0, fmt.Errorf("%v unsupported", i)
}

func main() {

		for _, exp := range tests {
			if r, err := parseAndEval(exp); err == nil {
				fmt.Println(exp, "=", r)
			} else {
				fmt.Printf("%s: %v\n", exp, err)
			}
		}
}
*/
