# DSync — Guia Completo do Projeto e de Go

> Documento de apoio para implementação e estudo. Serve como PRD atualizado + roteiro de aprendizado. Vamos marcar decisões fechadas com ✅ e pendências com 🔸, igual ao PRD original.

---

## 1. Visão geral do projeto

**DSync** (Dias Sincronizador) é um scheduler de tarefas recorrentes. Uma task tem uma **ação** e um **gatilho** — o gatilho pode ser um **intervalo** de repetição (ex: "a cada 30 minutos") ou um **horário fixo estilo cron** (ex: "toda segunda às 9h"). A ação pode ser:

- um **comando de shell**,
- a **cópia periódica de um arquivo**,
- a **compilação e execução de um código Go**.

### Problema que resolve
Hoje, agendar tarefas recorrentes exige `cron` (Linux/macOS) ou Task Scheduler (Windows), cada um com sintaxe e comportamento próprios. DSync propõe uma ferramenta única e portátil, que cobre tanto repetição por intervalo quanto horário fixo, com sintaxe consistente entre SOs.

> ✅ **Atualização de escopo:** na v0.1 do PRD, agendamento por horário fixo estava explicitamente fora de escopo. Decidimos incluir, com suporte a expressão cron completa (minuto, hora, dia do mês, mês, dia da semana) — ver decisão na seção 2.

### Escopo
- **Fase 1:** CLI (`dsync`) + daemon (`dsyncd`) rodando o agendador, com gatilhos por intervalo **e** por horário fixo (cron).
- **Fase 2:** aplicação de bandeja (tray), cliente visual do mesmo daemon.

### Fora de escopo
- Execução distribuída.
- Interface web.
- Múltiplos usuários/perfis.

### Glossário
| Termo | Significado |
|---|---|
| **Task** | Unidade de trabalho agendada: tem nome, ação e gatilho. |
| **Ação** | O que a task faz: rodar comando, copiar arquivo, ou compilar+rodar Go. |
| **Gatilho** | O que decide quando a task dispara: pode ser um Intervalo ou um Horário Fixo. |
| **Intervalo** | Um tipo de gatilho: duração entre execuções (`time.Duration`), contagem relativa. |
| **Horário fixo (cron)** | Outro tipo de gatilho: expressão de 5 campos (`minuto hora dia-do-mês mês dia-da-semana`) que descreve instantes específicos do relógio. |
| **Daemon (`dsyncd`)** | Processo em background que conta o tempo e dispara as tasks. |
| **IPC** | Comunicação entre processos — como a CLI fala com o daemon. |
| **PID file** | Arquivo com o número do processo do daemon, usado para saber se ele está vivo. |

---

## 2. Decisões já fechadas

| Decisão | Escolha | Por quê |
|---|---|---|
| Arquitetura do daemon | ✅ **Opção B** — comando explícito `dsync daemon start\|stop\|status` | Estado sempre explícito e previsível; melhor pra debugar e pra aprender ciclo de vida de processo. Se um comando qualquer rodar sem o daemon de pé, a CLI mostra mensagem amigável (`daemon não está rodando — rode 'dsync daemon start'`) em vez de erro de conexão recusada. |
| Parada do daemon | ✅ Via mensagem de "shutdown" pelo próprio canal IPC (não via sinal de SO) | Sinais não existem de forma equivalente no Windows; usar o mesmo canal que já existe evita reimplementar sinalização por SO. |
| Suporte a horário fixo | ✅ Sim, estilo **cron completo** (minuto, hora, dia do mês, mês, dia da semana) | Cobre casos como "todo dia às 12h" e "toda segunda às 9h", não só hora do dia isolada. |
| Diferenciação na CLI | ✅ Flag nova `--at "<expressão cron>"`. O argumento posicional de intervalo continua existindo e significa só intervalo. | Evita ambiguidade entre `12:00:00` como intervalo e como horário — cada sintaxe tem seu próprio parâmetro. |
| Modelo de dados | ✅ Generalizado desde o Módulo 1: `Task` passa a ter um campo `Trigger` (struct própria), em vez de `Interval` direto. | Intervalo e horário fixo calculam "próxima execução" de formas totalmente diferentes — melhor já nascer com essa separação do que refatorar depois. |
| Nomenclatura do código | ✅ Nomes de tipo/campo em **inglês** (`Task`, `ActionType`, `Trigger`, `NextRunning`...), comentários e este guia continuam em **português**. | Consistência com a convenção idiomática de Go e com libs de terceiros (ex: tags `json:"..."`), mantendo o aprendizado documentado em PT-BR nos comentários — ver nota no Módulo 1. |
| CLI | ✅ `cobra` (não parsing manual de `os.Args`) — já implementado (`cmd/root.go`, `cmd/task.go`, `cmd/task_add.go`). | Decisão do Módulo 3 antecipada; ensina o padrão real usado por CLIs Go do mercado. |
| Persistência de tasks | ✅ `TaskManager` (`internal/task/task_manager.go`) — singleton via `GetInstance()`, escrita atômica (arquivo `.tmp` + `os.Rename`), protegida por `sync.RWMutex`. | Trouxe a segurança de concorrência planejada pro Módulo 5 já pro Módulo 1 — evita reescrever a camada de persistência mais tarde. |
| Config centralizada | ✅ `internal/app.Config{TasksPath}` + `GetDsyncPath()` resolvem e criam `~/.dsync` automaticamente. | Um único lugar resolve onde tudo mora em disco (config e tasks), em vez de cada pacote calcular o caminho na mão. |
| Parsing da expressão cron | ✅ Escrito à mão (não usar `robfig/cron`) — módulo dedicado (2b) | Decisão consciente de aprendizado: parsing de campos com `*`, listas, ranges e steps, mais o cálculo de "próximo instante válido", é um exercício rico por si só. |

---

## 3. Perguntas ainda em aberto

Vamos decidir cada uma quando chegarmos no módulo correspondente — não precisa responder agora.

1. **Destino da cópia** (tasks tipo "arquivo"): flag `--to <destino>` no `task add`, ou diretório padrão configurável (ex: `~/.dsync/copies/<nome>/`)? *(Módulo 3/4)*
2. **"Código Go"**: aceita só caminho pra um arquivo `.go`, ou também um diretório/pacote? *(Módulo 4)*
3. 🔸 **`internal/trigger.ConvertTime` hoje valida como horário de relógio (H 0–23, M/S 0–59, CC 0–99), não como duração livre.** Isso parece alimentar melhor o Módulo 2b (horário fixo) do que o 2a (intervalo, que na v0.1 previa `HHHH` até 9999h). Precisa decidir: o parser de intervalo continua sendo uma função separada (aceitando `HHHH` grande), ou o conceito de "intervalo" também vira um horário limitado a 24h? *(Módulo 2)*
4. Task que falha repetidamente: continua tentando pra sempre ou desativa depois de N falhas? *(Módulo 5)*
5. Versionamento de cópias de arquivo: sobrescreve sempre, ou mantém histórico? *(Módulo 4)*
6. `task remove` de uma task em execução: espera terminar ou interrompe? *(Módulo 5)*
7. Biblioteca de tray pra Go e estratégia de empacotamento por SO. *(Módulo 8)*
8. `cmd/task_add.go` ainda não lê argumentos nem chama `TaskManager.Add` — é só um placeholder do `cobra`. Falta decidir a sintaxe final de flags antes de implementar de verdade. *(Módulo 3)*

---

## 4. Roadmap de implementação, por módulo

Cada módulo tem: objetivo, entregável concreto, conceitos novos de Go, e pacotes da stdlib envolvidos. A ordem importa — cada módulo depende do anterior.

### Módulo 0 — Setup e fundamentos
**Objetivo:** ambiente pronto e vocabulário mínimo de Go.
**Entregável:** projeto com `go.mod`, um `main.go` que compila e roda.
**Conceitos:** `go mod init`, `package main`, `func main()`, `go run` vs `go build`.

```bash
mkdir dsync && cd dsync
go mod init dsync
```

---

### Módulo 1 — Modelo de dados + persistência ✅ *(concluído — struct e persistência atualizadas, ver nota abaixo)*
**Objetivo:** representar uma `Task` em memória e salvar/carregar em disco.
**Entregável real (`internal/task`):**
```go
func NewTaskManager(path string) (*TaskManager, error)
func GetInstance() (*TaskManager, error)          // singleton, resolve ~/.dsync/tasks.json
func (tm *TaskManager) Add(t Task) error           // grava com lock + escrita atômica
func (tm *TaskManager) List() []Task
```
**Regras implementadas:**
- Arquivo em `~/.dsync/tasks.json`, resolvido por `app.GetDsyncPath()` (`os.UserHomeDir` + `os.MkdirAll`, ver `internal/app/Config.go`).
- `Add`/`saveLocked` gravam em `tasks.json.tmp` e usam `os.Rename` pra escrita atômica — já protegido por `sync.RWMutex`, adiantado do Módulo 5.
- Carregar sem arquivo existente → mapa vazio, sem erro (`os.IsNotExist`).
- `json.MarshalIndent` pra deixar o arquivo legível.

**Conceitos novos:** structs, structs aninhadas, tags de struct (`json:"..."`), ponteiros para valores opcionais (`*time.Time`), pacotes `os`, `encoding/json`, `path/filepath`, `time`, `sync.RWMutex`, `sync.Once` (usado no singleton `GetInstance`).

**Struct real** (código em inglês; comentários no arquivo continuam em português):
```go
type ActionType string

const (
	ActionCommand ActionType = "command"
	ActionFile    ActionType = "file"
	ActionGo      ActionType = "go"
)

type TriggerType string

const (
	TriggerInterval TriggerType = "interval"
	TriggerFixed    TriggerType = "fixed_time"
)

// Trigger decide QUANDO a task dispara. Só um dos dois campos de valor
// é preenchido, dependendo de Type.
type Trigger struct {
	Type     TriggerType   `json:"type"`
	Interval time.Duration `json:"interval,omitempty"` // usado se Type == interval
	CronExpr string        `json:"cron_expr,omitempty"` // usado se Type == fixed_time, ex: "0 12 * * *"
}

type Task struct {
	Name        string     `json:"name"`
	ActionType  ActionType `json:"action_type"`
	Payload     string     `json:"payload"`
	Destination string     `json:"destination,omitempty"`
	Trigger     Trigger    `json:"trigger"`
	NextRunning time.Time  `json:"next_running"`
	LastRunning *time.Time `json:"last_running,omitempty"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
}
```

**O que mudou em relação à primeira versão do PRD:**
- Todos os nomes de tipo/campo viraram inglês (`Nome`→`Name`, `TipoAcao`→`ActionType`, `Gatilho`→`Trigger`, `ProximaExecucao`→`NextRunning`, `UltimaExecucao`→`LastRunning`, `CriadaEm`→`CreatedAt`, `Destino`→`Destination`) — decisão fechada na seção 2. Os comentários no código-fonte continuam em português.
- `SaveTasks`/`LoadTasks` como funções soltas (ideia original) foram substituídas por `TaskManager`, que já resolve concorrência e escrita atômica — ver `internal/task/task_manager.go`.
- `Interval time.Duration` saiu direto da `Task` e entrou dentro da nova struct `Trigger`, junto com `CronExpr`. Isso é um exemplo de **struct aninhada** (uma struct como campo de outra) — comum em Go pra agrupar dados relacionados.

**Nota para depois:** `time.Duration` serializa em JSON como número de nanossegundos (feio de ler). Corrigir isso exige `MarshalJSON`/`UnmarshalJSON` customizados — ótimo exercício de interfaces implícitas (ver seção 5.6), mas não é bloqueante agora.

---

### Módulo 2 — Parsing de gatilhos: intervalo (2a) e cron (2b)
Esse módulo cresceu — agora cobre os dois tipos de `Trigger`. São dois sub-módulos independentes; dá pra fazer 2a e já seguir pro Módulo 3 antes de encarar o 2b, se preferir avançar e voltar depois.

#### 2a — Parsing de horário 🔸 *(implementado, mas ver pendência #3 da seção 3)*
**Entregável real:** `func ConvertTime(s string) (*clock.Time, error)` (`internal/trigger/trigger_time.go`, tipo `Time` em `internal/clock/time.go`).

```go
// internal/clock/time.go
package clock

type Time struct {
	Hours        int8
	Minutes      int8
	Seconds      int8
	CentiSeconds int8
}
```

Aceita `H:M:S` (3 partes) ou `H:M:S:CC` (4 partes), separadas por `:` — não exige zero à esquerda. Validação real:

| Campo | Faixa aceita |
|---|---|
| Horas | 0–23 |
| Minutos | 0–59 |
| Segundos | 0–59 |
| Centésimos (`CC`, opcional) | 0–99 |

Isso é uma validação de **horário de relógio de 24h**, diferente da ideia original do PRD (intervalo/duração com `HHHH` até 9999h). Ver pendência #3 na seção 3 — hoje `ConvertTime` serve melhor de base pro Módulo 2b (horário fixo) do que pro 2a (intervalo) como descrito originalmente.

**Conceitos novos usados:** `strings.Split`, `strconv.ParseInt`, criação de `error` customizado (`fmt.Errorf`), **table-driven tests** com o pacote `testing` — já implementado de verdade em `internal/trigger/trigger_time_test.go`.

**Exemplo de teste table-driven** (mesmo padrão do arquivo real, versão didática):
```go
func TestParseIntervalo(t *testing.T) {
	casos := []struct {
		entrada string
		esperado time.Duration
		erroEsperado bool
	}{
		{"00:00:10", 10 * time.Second, false},
		{"01:00:00", time.Hour, false},
		{"99:99:00", 0, true}, // MM inválido
	}
	for _, c := range casos {
		resultado, err := ParseIntervalo(c.entrada)
		if c.erroEsperado && err == nil {
			t.Errorf("esperava erro para %q", c.entrada)
		}
		if resultado != c.esperado {
			t.Errorf("entrada %q: esperado %v, obtido %v", c.entrada, c.esperado, resultado)
		}
	}
}
```

#### 2b — Parsing da expressão cron (estilo cron completo, escrito à mão) 🔸 *(não iniciado)*
**Objetivo:** dado `--at "0 12 * * *"`, entender a expressão e saber calcular o próximo instante que ela descreve a partir de agora.
**Entregável:**
```go
func ParseCron(expr string) (CronSchedule, error)
func (c CronSchedule) Proxima(a partir time.Time) time.Time
```
Esse módulo tem duas partes bem distintas, e vale tratar cada uma separadamente:

**Parte 1 — parsing dos 5 campos.** Uma expressão cron tem a forma `minuto hora dia-do-mês mês dia-da-semana`:

| Campo | Faixa | Exemplo de sintaxe |
|---|---|---|
| minuto | 0–59 | `*`, `30`, `*/15`, `0,15,30,45` |
| hora | 0–23 | `12`, `9-17` |
| dia do mês | 1–31 | `*`, `1`, `1,15` |
| mês | 1–12 | `*`, `6`, `1-6` |
| dia da semana | 0–6 (0 = domingo) | `*`, `1`, `1-5` |

Cada campo aceita 4 formas de sintaxe, e seu parser precisa reconhecer todas:
- `*` → qualquer valor é válido pra esse campo.
- valor único → `12`.
- lista → `1,15,30` (`strings.Split` por vírgula).
- range → `9-17` (do 9 ao 17, inclusive).
- step → `*/15` (a cada 15 unidades, começando do 0) — pode combinar com range: `10-40/5`.

Uma boa representação em memória pra cada campo é um `map[int]bool` ou `[]bool` (um "calendário" de quais valores são aceitos) — assim, na hora de checar "esse minuto bate?", vira só uma consulta O(1), em vez de reprocessar a string toda vez.

```go
type CronSchedule struct {
	Minutos      map[int]bool
	Horas        map[int]bool
	DiasDoMes    map[int]bool
	Meses        map[int]bool
	DiasDaSemana map[int]bool
}
```

**Parte 2 — calcular a próxima execução.** Dado um `CronSchedule` e o instante atual, o algoritmo mais simples (e didático) é: avance minuto a minuto a partir de agora, e pra cada minuto candidato, cheque se ele bate com todos os 5 campos (`Minutos[m.Minute()]`, `Horas[m.Hour()]`, etc.). O primeiro que bater em tudo é a resposta. Não é o algoritmo mais eficiente do mundo (dá pra otimizar depois pulando direto pro próximo valor válido de cada campo), mas é o mais fácil de entender e testar primeiro — otimização prematura aqui só atrapalha o aprendizado.

⚠️ **Pegadinha real de calendário:** "dia do mês" e "dia da semana" são combinados com **OR**, não **AND**, na convenção clássica do cron — `"0 0 1,15 * 1"` significa "todo dia 1 e 15 do mês, **ou** toda segunda-feira", não "só quando dia 1/15 cair numa segunda". Vale decidir se você quer seguir essa convenção (mais compatível com cron de verdade) ou simplificar pra AND (mais intuitivo, mas diferente do cron tradicional) — sua escolha, mas documente no código qual foi.

**Conceitos novos:** parsing de string mais elaborado, `map[int]bool` como estrutura de "conjunto", métodos com receiver (`func (c CronSchedule) Proxima(...)`), iteração de `time.Time` (`t.Add(time.Minute)`, `t.Minute()`, `t.Weekday()`), testes table-driven com casos de borda (virada de mês, virada de ano, fevereiro em ano bissexto).

---

### Módulo 3 — CLI 🔸 *(iniciado — esqueleto cobra existe, comandos ainda placeholder)*
**Objetivo:** comandos `dsync task add|remove|ls|execute`.
**Estado atual:** `cobra` já escolhido e em uso (decisão da seção 2). Existe `rootCmd` (`cmd/root.go`, `Use: "dsync"`) e `dsync task` (`cmd/task.go`), com subcomando `dsync task add` (`cmd/task_add.go`) — mas o `Run` de `task add` hoje só imprime `"taskAdd called"`, sem ler flags/args nem chamar `tasks.GetInstance().Add(...)`. Falta implementar de verdade.
**Conceitos novos:** parsing de argumentos com `cobra` (flags, subcomandos, `PersistentFlags`), formatação de tabela (`text/tabwriter`, ainda não usado), detecção de tipo de ação (caminho existente vs `.go` vs comando).

**Sintaxe do `add` com os dois tipos de gatilho (alvo, ainda não implementado):**
```bash
dsync task add "echo bom dia" 00:00:10                 # gatilho: intervalo
dsync task add "backup.sh" --at "0 12 * * *"           # gatilho: horário fixo (cron)
```
O argumento posicional de intervalo e a flag `--at` são **mutuamente exclusivos** — se os dois vierem juntos, é erro de validação (mensagem clara, não silenciosamente ignora um dos dois).

---

### Módulo 4 — Execução das ações 🔸 *(não iniciado)*
**Objetivo:** implementar de fato o que cada tipo de task faz.
**Entregável:** `func Executar(t Task) error` cobrindo os 3 tipos.

| Tipo | Como executa |
|---|---|
| Comando | `exec.Command("sh", "-c", cmd)` no Unix / `exec.Command("cmd", "/C", cmd)` no Windows |
| Arquivo | `io.Copy` de origem pro destino |
| Código Go | `exec.Command("go", "build", ...)` na 1ª vez ou se hash mudou; depois só roda o binário em cache |

**Conceitos novos:** `os/exec`, captura de `stdout`/`stderr` (`cmd.Output()` vs `cmd.CombinedOutput()`), `io.Copy`, hashing de arquivo (`crypto/sha256`) pra detectar mudança no `.go`, `context.Context` pra timeout de execução.

---

### Módulo 5 — Scheduler (o coração concorrente) 🔸 *(não iniciado — parte de concorrência já adiantada no Módulo 1)*
**Objetivo:** processo que mantém timers rodando e dispara tasks nos intervalos certos.
**Entregável:** loop de agendamento rodando em background dentro do daemon.
**Conceitos novos — os mais "Go" do projeto:**
- **goroutines** (`go func() {...}()`) — uma por task ativa, ou um loop central com `time.Ticker`.
- **channels** — comunicação entre a goroutine do scheduler e o resto do programa (ex: canal de "task disparou", canal de "pare tudo").
- **`sync.Mutex`** — já usado (via `sync.RWMutex`) no `TaskManager` do Módulo 1; aqui se estende pro loop do scheduler.
- **graceful shutdown** — capturar `SIGINT`/`SIGTERM` (`os/signal`) e desligar limpo, salvando estado antes de sair.
- Política de concorrência por task: se o intervalo disparar de novo enquanto a execução anterior ainda roda, **pula e loga aviso** (evita empilhar).

---

### Módulo 6 — Daemon + IPC 🔸 *(não iniciado)*
**Objetivo:** processo `dsyncd` separado, e a CLI conversando com ele.
**Entregável:** `dsync daemon start|stop|status` funcionando; `task add/remove/ls/execute` passam a falar com o daemon em vez de mexer direto no arquivo.
**Conceitos novos:** unix socket (`net.Listen("unix", ...)`) no Linux/macOS, named pipe no Windows, ou HTTP local como alternativa mais simples (`net/http` num `127.0.0.1:porta`), serialização de mensagens (JSON de novo), PID file, checagem de processo vivo.

**Nota de design:** por simplicidade multiplataforma, pode valer a pena começar com **HTTP local** (mais fácil, mesma API em qualquer SO) e só migrar pra unix socket/named pipe depois, se quiser otimizar.

---

### Módulo 7 — Logs 🔸 *(não iniciado)*
**Objetivo:** cada task grava histórico de execução.
**Entregável:** `~/.dsync/logs/<nome>.log` com timestamps, stdout/stderr, código de saída.
**Conceitos novos:** `log/slog` (já usado hoje só pra log de console em `main.go`/`internal/app`; falta o modo arquivo), abrir arquivo em modo append (`os.OpenFile` com `os.O_APPEND`).

---

### Módulo 8 — Tray app *(Fase 2)*
**Objetivo:** cliente visual do mesmo `dsyncd`, sem duplicar lógica de agendamento.
**Entregável:** ícone de bandeja com menu (listar, executar agora, pausar/retomar, remover, abrir logs, configurações, sair).
**Conceitos novos:** binding de lib externa (`getlantern/systray` ou `fyne.io/systray`), reaproveitamento do cliente IPC feito no Módulo 6.
**Pendência:** biblioteca definitiva e empacotamento (.deb/.dmg/.msi) — decidir mais perto da hora.

---

## 5. Fundamentos de Go — guia de referência

Você já conhece `var`, `func`, `if`/`for`. Aqui está o que falta pra cobrir tudo que o roadmap acima vai exigir. Não precisa ler tudo de uma vez — volte aqui quando o módulo correspondente pedir o conceito.

### 5.1 Tipos e zero values
Toda variável em Go tem um valor padrão quando declarada sem inicialização — não existe "undefined":
```go
var i int     // 0
var s string  // ""
var b bool    // false
var t Task    // struct com todos os campos no zero value
```
Isso importa pro DSync: por exemplo, o zero value de `time.Time` é `0001-01-01 00:00:00`, não "vazio" — por isso usamos `*time.Time` (ponteiro) quando queremos representar "ainda não aconteceu" (`Task.LastRunning`).

### 5.2 Structs e métodos
Struct é um tipo composto — agrupa campos relacionados (como o `Task`). Métodos são funções "presas" a um tipo, via **receiver**:
```go
func (t Task) EstaAtrasada() bool {
	return time.Now().After(t.NextRunning)
}
```
`(t Task)` é o receiver — dentro do método, `t` é uma cópia da task. Se o método precisa **modificar** a struct original, o receiver tem que ser um ponteiro:
```go
func (t *Task) MarcarComoErro() {
	t.Status = "erro"
}
```
Regra prática: se o método muda algo, use `*Task`. Se só lê, pode usar `Task` (mas por convenção, se qualquer método do tipo usa ponteiro, os outros métodos do mesmo tipo também usam, por consistência).

### 5.3 Ponteiros
`*T` é "ponteiro para T" (um endereço de memória), `&x` pega o endereço de `x`, `*p` "desreferencia" (pega o valor apontado):
```go
var x int = 5
p := &x       // p é *int, aponta pra x
*p = 10       // agora x vale 10
```
No DSync isso aparece em dois lugares principais: `*time.Time` para "opcional" (`LastRunning`), e receivers de método (`*Task`) para mutação.

### 5.4 Slices, arrays e maps
- **Array**: tamanho fixo, raramente usado direto.
- **Slice**: array dinâmico — o que você vai usar sempre. `[]Task` é uma slice de tasks (é o que `TaskManager.List()` retorna).
- **Map**: dicionário chave→valor. É como o `TaskManager` indexa tasks por nome: `map[string]Task`.

```go
var tasks []Task                    // slice vazia
tasks = append(tasks, novaTask)     // adiciona

porNome := make(map[string]Task)
porNome["backup-diario"] = tasks[0]
t, existe := porNome["backup-diario"] // segundo valor: existe ou não
```

### 5.5 Tratamento de erro
Go não tem exceptions. Funções que podem falhar retornam um `error` como último valor, e você checa explicitamente:
```go
func (tm *TaskManager) load() error {
	dados, err := os.ReadFile(tm.path)
	if err != nil {
		return fmt.Errorf("erro lendo tasks: %w", err)
	}
	// ...
}
```
`%w` (em vez de `%v`) "envolve" o erro original, preservando a cadeia — quem recebe pode usar `errors.Is`/`errors.As` pra inspecionar a causa raiz. Isso importa bastante no Módulo 4, onde erros de execução de comando/compilação precisam chegar até o log com contexto.

### 5.6 Interfaces (implementação implícita)
Uma interface define um conjunto de métodos. Qualquer tipo que implemente esses métodos "satisfaz" a interface automaticamente — **sem declarar isso explicitamente** (diferente de Java/C#):
```go
type error interface {
	Error() string
}
```
Qualquer tipo com um método `Error() string` já é um `error`. Isso é a base de como `encoding/json` permite customizar serialização: se seu tipo tem `MarshalJSON() ([]byte, error)`, o pacote `json` usa esse método automaticamente. É assim que resolveríamos o problema do `time.Duration` feio no JSON (mencionado no Módulo 1).

### 5.7 Goroutines e channels
Goroutine é uma "thread leve" gerenciada pelo runtime do Go:
```go
go func() {
	fmt.Println("rodando em paralelo")
}()
```
Channel é um cano tipado pra goroutines se comunicarem, em vez de compartilhar memória diretamente:
```go
ch := make(chan string)
go func() {
	ch <- "task disparou"
}()
msg := <-ch // bloqueia até chegar algo
```
No Módulo 5, o scheduler provavelmente terá uma goroutine central com `time.Ticker`, e channels para sinalizar "pare" (shutdown) ou "task X disparou agora".

### 5.8 sync.Mutex / sync.RWMutex
Quando múltiplas goroutines (ou o daemon + a CLI) podem ler/escrever a mesma coisa (a lista de tasks) ao mesmo tempo, é preciso proteger com um lock. É exatamente o que `TaskManager` já faz hoje:
```go
type TaskManager struct {
	mu    sync.RWMutex
	tasks map[string]Task
	path  string
}

func (tm *TaskManager) Add(t Task) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.tasks[t.Name] = t
	return tm.saveLocked()
}
```
`RWMutex` permite várias leituras simultâneas (`RLock`, usado em `List()`), mas só uma escrita por vez (`Lock`, usado em `Add()`), e escrita bloqueia leitura. `defer` executa a chamada no final da função, garantindo que o `Unlock` acontece mesmo se a função retornar mais cedo por erro.

### 5.9 defer, panic, recover
- `defer` adia uma chamada até o final da função (fecho de arquivo, unlock de mutex, etc.) — muito usado.
- `panic`/`recover` são para erros irrecuperáveis/bugs, não para fluxo normal de erro (isso é `error`, como na seção 5.5). Hoje o projeto usa `panic` em `internal/app.GetDsyncPath()` pra falhas de setup irrecuperáveis (`os.UserHomeDir` ou I/O real ao checar `~/.dsync`) — falha de configuração de ambiente não tem um jeito sensato de "continuar", então faz sentido parar cedo. Fora isso, é bom usar pouco, exceto talvez pra evitar que uma task travando derrube o daemon inteiro.

### 5.10 Pacotes e módulos
- Um **módulo** (`go.mod`) é o projeto inteiro; pode conter vários **pacotes** (pastas com `.go` dentro).
- `package main` é especial: é o ponto de entrada executável.
- **`internal/`** é um nome de diretório especial reconhecido pelo próprio compilador do Go: qualquer pacote dentro dele só pode ser importado por código do mesmo módulo (aqui, `github.com/Alangdp/dsync.git`) — é assim que o Go garante que "detalhe de implementação" não vaza pra quem importar o projeto de fora.

**Estrutura atual do repositório:**
```
dsync/
├── go.mod
├── Makefile              (make build → dsync.exe)
├── main.go               (entrypoint da CLI, package main)
├── cmd/                   (package cmd — cobra: root, task, task add)
│   ├── root.go
│   ├── task.go
│   └── task_add.go
└── internal/
    ├── app/                (Config, GetDsyncPath — resolve ~/.dsync)
    ├── clock/               (Time: representação de horário HH:MM:SS:CC)
    ├── filesystem/           (Exists)
    ├── task/                  (Task, Trigger, TaskManager)
    └── trigger/                (ConvertTime)
```
Quando o Módulo 6 (daemon) for implementado, a convenção é `cmd/dsyncd/main.go` como segundo `package main` — ainda não existe.

### 5.11 os/exec — rodando processos externos
```go
cmd := exec.Command("sh", "-c", "echo bom dia")
saida, err := cmd.CombinedOutput() // stdout + stderr juntos
```
`cmd.Output()` só stdout (erro se stderr tiver algo e o processo falhar); `cmd.CombinedOutput()` junta os dois — geralmente o que você quer pra log.

### 5.12 context.Context — timeout e cancelamento
```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
cmd := exec.CommandContext(ctx, "sh", "-c", comandoLongo)
```
Se o comando não terminar em 30s, o `ctx` cancela e mata o processo. Central pro Módulo 4 (timeout de execução de task).

### 5.13 Testes em Go
Arquivo `*_test.go`, função `func TestAlgo(t *testing.T)`, roda com `go test ./...`. Padrão table-driven já mostrado na seção do Módulo 2 — vale adotar em todo o projeto, é o estilo idiomático de Go. Mensagens de falha de teste (`t.Errorf`, `t.Fatalf`) devem ficar em inglês, junto com o resto da saída do programa — só os comentários continuam em português.

### 5.14 Cross-platform (Linux/macOS/Windows no mesmo binário)
- **Build tags / sufixo de arquivo**: `daemon_unix.go` vs `daemon_windows.go` — o compilador escolhe o arquivo certo pelo nome (`_windows.go`, `_linux.go`, `_darwin.go`) ou por diretiva `//go:build windows` no topo do arquivo.
- **`SysProcAttr`**: campo de `exec.Cmd` que varia por SO — é onde as diferenças de "processo desanexado" (Módulo 6) aparecem na prática.
- Isso vai ficar mais concreto quando chegarmos no daemon — por enquanto só fica de referência.

---

## 6. Referência rápida de pacotes usados no projeto

| Pacote | Pra que serve no DSync |
|---|---|
| `os` | home dir, args, arquivos, sinais |
| `encoding/json` | serializar/desserializar tasks e config |
| `path/filepath` | montar caminhos multiplataforma |
| `time` | duração, timestamps |
| `strings`, `strconv` | parsing de horário (`ConvertTime`) |
| `errors`, `fmt` | erros customizados |
| `testing` | testes |
| `sync` | `RWMutex`/`Once` — `TaskManager` e o singleton `GetInstance` |
| `log/slog` | logging estruturado (já em uso em `main.go`/`internal/app`) |
| `github.com/spf13/cobra` | parsing de comandos/flags da CLI (não é stdlib — dependência externa, ver Módulo 3) |
| `os/exec` | rodar comando/compilar Go *(Módulo 4, ainda não usado)* |
| `io` | cópia de arquivo *(Módulo 4, ainda não usado)* |
| `crypto/sha256` | hash do `.go` pra saber se recompila *(Módulo 4, ainda não usado)* |
| `context` | timeout de execução *(Módulo 4, ainda não usado)* |
| `os/signal` | shutdown gracioso *(Módulo 5, ainda não usado)* |
| `net` / `net/http` | IPC CLI↔daemon *(Módulo 6, ainda não usado)* |
| `text/tabwriter` | saída em tabela do `task ls` *(Módulo 3, ainda não usado)* |

---

## 7. Próximos passos imediatos

1. ✅ Módulo 1 concluído: `Task`/`Trigger` em inglês, persistência via `TaskManager` com escrita atômica.
2. 🔸 Resolver a pendência #3 da seção 3: decidir se `internal/trigger.ConvertTime`/`internal/clock.Time` é o parser do Módulo 2b (horário fixo) ou se o Módulo 2a (intervalo/duração) ainda precisa de uma função separada.
3. Terminar o Módulo 3: fazer `cmd/task_add.go` de fato ler flags/args e chamar `tasks.GetInstance().Add(...)`, e implementar `task ls`/`task remove`.
4. Módulo 2b (parser de cron) pode vir logo depois ou ser intercalado com o resto do Módulo 3 — como preferir.

---

*Documento vivo — atualizar conforme decisões forem fechadas e módulos avançarem.*
