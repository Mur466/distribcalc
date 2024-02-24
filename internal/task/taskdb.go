package task

import (
	"context"
	"encoding/json"
	"fmt"

	"go.uber.org/zap"

	"github.com/Mur466/distribcalc/internal/db"
	l "github.com/Mur466/distribcalc/internal/logger"
)

// атомарно выбираем ожидающую операцию и переводим ее в процесс, а также и task, если нужно
func (n *Node) SetToProcess(agent_id string) {
	t := Tasks[n.Task_id]
	t.SetNodeStatus(n.Node_id, "in progress", NodeStatusInfo{Agent_id: agent_id})
	done := 0
	for _, n2 := range t.TreeSlice {
		if n2.Status == "done" {
			done++
		}
	}
	t.Message = fmt.Sprintf("Nodes complete %v of %v", done, len(t.TreeSlice))
	if t.Status != "in progress" {
		t.SetStatus("in progress", TaskStatusInfo{})
	} else {
		t.SaveTask()
	}

}

func (t *Task) SaveTask() {

	//  готовим json
	data, err := json.Marshal(t)
	if err != nil {
		l.Logger.Error("Error on marshalling ",
			zap.String("error", err.Error()),
			zap.String("data", string(fmt.Sprintf("%+v", t))),
		)
		return
	}

	need_update := true
	if t.Task_id < 0 {
		// новое выражение
		need_update = false
		err := db.Conn.QueryRow(context.Background(), `
			INSERT INTO tasks (data) VALUES ($1) returning task_id;
			`, data).Scan(&t.Task_id)
		if err != nil {
			l.Logger.Error("Error on insert to TASKS",
				zap.String("error", err.Error()),
				zap.Int("Task_id", t.Task_id),
				zap.String("data", string(data)),
			)
			return
		}
		if len(t.TreeSlice) > 0 {
			//выражение еще не было сохранено, но у него есть узлы
			// такое возможно, если при создании выражения БД была недоступна
			// теперь мы получили task_id из новой записи в БД
			// поправим task_id у нод, ведь на момент их создания task_id был неизвестен (-1)
			for _, n := range t.TreeSlice {
				n.Task_id = t.Task_id
			}
			// теперь надо обновить данные в  БД
			need_update = true
			data, err = json.Marshal(t)
			if err != nil {
				l.Logger.Error("Error on marshalling ",
					zap.String("error", err.Error()),
					zap.String("data", string(fmt.Sprintf("%+v", t))),
				)
				return
			}
		}
	}
	if need_update {
		// обновляем
		_, err := db.Conn.Exec(context.Background(), `
			UPDATE tasks 
			  SET data=$1
			WHERE task_id=$2;
			`, data, t.Task_id)
		if err != nil {
			l.Logger.Error("Error on update to TASKS",
				zap.String("error", err.Error()),
				zap.Int("Task_id", t.Task_id),
				zap.String("data", string(data)),
			)
			return
		}
	}
	l.Logger.Debug("Inserted/Updated TASKS",
		zap.Int("Task_id", t.Task_id),
		zap.String("data", string(data)),
	)

}

func (t *Task) DeleteTask() {
	_, err := db.Conn.Exec(context.Background(), `
	DELETE from tasks 
	WHERE task_id=$1;
	`, t.Task_id)
	if err != nil {
		l.Logger.Error("Error on delete from TASKS",
			zap.String("error", err.Error()),
			zap.Int("Task_id", t.Task_id),
		)
	}
}

func InitTasks() {

	// todo Может сделать какой-то лимит? Всю таблицу в базу тянуть не комильфо
	rows, err := db.Conn.Query(context.Background(), `
				SELECT task_id, data
				FROM tasks
				ORDER BY task_id DESC;
				`)
	if err != nil {
		l.Logger.Error("Error on select from TASKS",
			zap.String("error", err.Error()),
		)
		return
	}
	for rows.Next() {
		task_id := -1
		task_data := []byte{}
		err := rows.Scan(&task_id, &task_data)
		if err != nil {
			l.Logger.Error("Error on fetch from TASKS",
				zap.String("error", err.Error()),
			)
			return
		}
		task := Task{}
		err = json.Unmarshal([]byte(task_data), &task)
		if err != nil {
			l.Logger.Error("Error unmarshall data on fetch from TASKS",
				zap.String("error", err.Error()),
				zap.Int("task_id", task_id),
				zap.String("data", string(task_data)),
			)
			continue // пропустим запись
		}

		// добавляем или подменяем задание
		Tasks[task_id] = &task
	}
	UpLostTasks()
}

// Читаем из базы задания
// limit и offset == 0 дают полный список
func ListTasks(limit int, offset int) []*Task {
	strlimit := "ALL"
	if limit > 0 {
		strlimit = fmt.Sprintf("%d", limit)
	}
	var tasks []*Task = make([]*Task, 0)

	rows, err := db.Conn.Query(context.Background(), fmt.Sprintf(`
				SELECT task_id, data
				FROM tasks
				ORDER BY task_id DESC
				LIMIT %s OFFSET %d
				;
				`, strlimit, offset))
	if err != nil {
		l.Logger.Error("Error on select from TASKS",
			zap.String("error", err.Error()),
		)
		return tasks
	}
	for rows.Next() {
		task_id := -1
		task_data := []byte{}
		err := rows.Scan(&task_id, &task_data)
		if err != nil {
			l.Logger.Error("Error on fetch from TASKS",
				zap.String("error", err.Error()),
			)
			return tasks
		}
		task := Task{}
		err = json.Unmarshal([]byte(task_data), &task)
		if err != nil {
			l.Logger.Error("Error unmarshall data on fetch from TASKS",
				zap.String("error", err.Error()),
				zap.Int("task_id", task_id),
				zap.String("data", string(task_data)),
			)
			continue // пропустим запись
		}
		// добавляем задание
		tasks = append(tasks, &task)
	}
	return tasks

}
