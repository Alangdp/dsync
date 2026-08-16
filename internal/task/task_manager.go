package tasks

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/Alangdp/dsync.git/internal/app"
)

type TaskManager struct {
	mu    sync.RWMutex
	tasks map[string]Task
	path  string
}

/*
Cria um novo gerenciador de tarefas
*/
func NewTaskManager(path string) (*TaskManager, error) {
	tm := &TaskManager{tasks: make(map[string]Task), path: path}
	if err := tm.load(); err != nil {
		return nil, err
	}
	return tm, nil
}

/*
Cria um novo gerenciador de tarefas
*/
func (tm *TaskManager) load() error {
	data, err := os.ReadFile(tm.path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	var list []Task
	if err := json.Unmarshal(data, &list); err != nil {
		return err
	}

	for _, t := range list {
		tm.tasks[t.Name] = t
	}
	return nil
}

/*
Retorna a lista de tarefas carregadas
*/
func (tm *TaskManager) List() []Task {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	out := make([]Task, 0, len(tm.tasks))
	for _, t := range tm.tasks {
		out = append(out, t)
	}
	return out
}

/*
Adicionada uma nova tarefa a lista
*/
func (tm *TaskManager) Add(t Task) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tm.tasks[t.Name] = t
	return tm.saveLocked()
}

/*
Salva o arquivos de tarefas de maneira concorrente
*/
func (tm *TaskManager) saveLocked() error {
	list := make([]Task, 0, len(tm.tasks))
	for _, t := range tm.tasks {
		list = append(list, t)
	}

	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}

	tmp := tm.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, tm.path)
}

var (
	instance *TaskManager
	once     sync.Once
	initErr  error
)

/*
Retorna uma nova instância singleton
*/
func GetInstance() (*TaskManager, error) {

	once.Do(func() {
		// Caminho do projeto
		configPath := app.GetDsyncPath()
		// Caminho das tasks
		tasksFile := filepath.Join(configPath, "tasks.json")
		// Cria uma nova instância do gerenciador de tarefas
		instance, initErr = NewTaskManager(tasksFile)
	})
	return instance, initErr
}
