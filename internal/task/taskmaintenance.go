package task

import (
	"time"
)


func UpLostTasks() {
	for _, t := range Tasks {
		if t.Status == "in progress" {
			// проверим нет ли зависших нод
			// если сервер был выключен, агенты могли прислать результаты, которые мы не получили и уже не получим
			for _, n := range t.TreeSlice {
				if n.Status == "in progress" {
					timeout := n.Date_start.Add(time.Duration(n.Operator_delay*2) * time.Second)
					if time.Now().After(timeout) {
						// если уже ждем вдвое дольше таймаута, подадим повторно
						t.SetNodeStatus(n.Node_id, "ready", NodeStatusInfo{})
					}
				}
				if n.Status == "waiting" {
					if (n.Child1_node_id == -1 || // нет дочки
						t.TreeSlice[n.Child1_node_id].Status == "done") &&
						(n.Child2_node_id == -1 || // нет дочки
						t.TreeSlice[n.Child2_node_id].Status == "done") {
						// дочек нет или они посчитаны, можем считать папу
						// вообще такое может быть только в случае багов или сбоев
						// на всякий случай запишем результаты дочек в папу
						if n.Child1_node_id != -1 {
							n.Operand1 = int(t.TreeSlice[n.Child1_node_id].Result)
						}
						if n.Child2_node_id != -1 {
							n.Operand2 = int(t.TreeSlice[n.Child2_node_id].Result)
						}
						// подаем в расчет
						t.SetNodeStatus(n.Node_id, "ready", NodeStatusInfo{})					
					}
				}
			}
		}
	}
}

