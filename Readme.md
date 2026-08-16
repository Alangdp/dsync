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
| Modelo de dados | ✅ Generalizado desde o Módulo 1: `Task` passa a ter um campo `Gatilho` (struct própria), em vez de `Intervalo` direto. | Intervalo e horário fixo calculam "próxima execução" de formas totalmente diferentes — melhor já nascer com essa separação do que refatorar depois. |
| Parsing da expressão cron | ✅ Escrito à mão (não usar `robfig/cron`) — módulo dedicado (2b) | Decisão consciente de aprendizado: parsing de campos com `*`, listas, ranges e steps, mais o cálculo de "próximo instante válido", é um exercício rico por si só. |

---

## 3. Perguntas ainda em aberto

Vamos decidir cada uma quando chegarmos no módulo correspondente — não precisa responder agora.

1. **Destino da cópia** (tasks tipo "arquivo"): flag `--to <destino>` no `task add`, ou diretório padrão configurável (ex: `~/.dsync/copies/<nome>/`)? *(Módulo 3/4)*
2. **"Código Go"**: aceita só caminho pra um arquivo `.go`, ou também um diretório/pacote? *(Módulo 4)*
3. `CC` no formato longo do intervalo (`HHHH:MM:SS:CC`) = centésimos de segundo — confirmar. *(Módulo 2)*
4. Task que falha repetidamente: continua tentando pra sempre ou desativa depois de N falhas? *(Módulo 5)*
5. Versionamento de cópias de arquivo: sobrescreve sempre, ou mantém histórico? *(Módulo 4)*
6. `task remove` de uma task em execução: espera terminar ou interrompe? *(Módulo 5)*
7. Biblioteca de tray pra Go e estratégia de empacotamento por SO. *(Módulo 8)*

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

### Módulo 1 — Modelo de dados + persistência *(em andamento — struct atualizada, ver nota abaixo)*
**Objetivo:** representar uma `Task` em memória e salvar/carregar em disco.
**Entregável:**
```go
func SaveTasks(tasks []Task) error
func LoadTasks() ([]Task, error)
```
**Regras:**
- Arquivo em `~/.dsync/tasks.json` (`os.UserHomeDir` + `os.MkdirAll`).
- `SaveTasks` sobrescreve o arquivo inteiro (sem lock de concorrência ainda — isso vem no Módulo 5).
- `LoadTasks` sem arquivo existente → lista vazia, sem erro.
- Usar `json.MarshalIndent` para o arquivo ficar legível.

**Conceitos novos:** structs, structs aninhadas, tags de struct (`json:"..."`), ponteiros para valores opcionais (`*time.Time`), pacotes `os`, `encoding/json`, `path/filepath`, `time`.

**Struct atualizada** (agora com `Gatilho` separado, pra suportar intervalo e horário fixo/cron):
```go
type TipoAcao string

const (
	AcaoComando TipoAcao = "comando"
	AcaoArquivo TipoAcao = "arquivo"
	AcaoGoCode  TipoAcao = "go"
)

type TipoGatilho string

const (
	GatilhoIntervalo   TipoGatilho = "intervalo"
	GatilhoHorarioFixo TipoGatilho = "horario_fixo" // estilo cron
)

// Gatilho decide QUANDO a task dispara. Só um dos dois campos de valor
// é preenchido, dependendo de Tipo — isso fica mais robusto ainda no
// Módulo 2, quando os dois viram tipos totalmente distintos.
type Gatilho struct {
	Tipo      TipoGatilho   `json:"tipo"`
	Intervalo time.Duration `json:"intervalo,omitempty"` // usado se Tipo == intervalo
	CronExpr  string        `json:"cron_expr,omitempty"` // usado se Tipo == horario_fixo, ex: "0 12 * * *"
}

type Task struct {
	Nome            string     `json:"nome"`
	TipoAcao        TipoAcao   `json:"tipo_acao"`
	Payload         string     `json:"payload"`
	Destino         string     `json:"destino,omitempty"`
	Gatilho         Gatilho    `json:"gatilho"`
	ProximaExecucao time.Time  `json:"proxima_execucao"`
	UltimaExecucao  *time.Time `json:"ultima_execucao,omitempty"`
	Status          string     `json:"status"`
	CriadaEm        time.Time  `json:"criada_em"`
}
```

**O que mudou em relação à primeira versão:**
- O antigo campo `Tipo TaskType` (comando/arquivo/go) virou `TipoAcao`, pra não confundir com o novo `TipoGatilho` (intervalo/horário fixo) — dois conceitos diferentes, dois nomes diferentes.
- `Intervalo time.Duration` saiu direto da `Task` e entrou dentro da nova struct `Gatilho`, junto com `CronExpr`. Isso é um exemplo de **struct aninhada** (uma struct como campo de outra) — comum em Go pra agrupar dados relacionados.
- 🔸 **Se você já tinha escrito `SaveTasks`/`LoadTasks` com a struct antiga**, o código em si não muda muito (continua sendo só `json.MarshalIndent`/`json.Unmarshal` num `[]Task`) — só ajuste os nomes de campo na hora de montar uma `Task` de teste.

**Nota para depois:** `time.Duration` serializa em JSON como número de nanossegundos (feio de ler). Corrigir isso exige `MarshalJSON`/`UnmarshalJSON` customizados — ótimo exercício de interfaces implícitas (ver seção 6.6), mas não é bloqueante agora.

---

### Módulo 2 — Parsing de gatilhos: intervalo (2a) e cron (2b)
Esse módulo cresceu — agora cobre os dois tipos de `Gatilho`. São dois sub-módulos independentes; dá pra fazer 2a e já seguir pro Módulo 3 antes de encarar o 2b, se preferir avançar e voltar depois.

#### 2a — Parsing do intervalo
**Objetivo:** converter `HH:MM:SS` e `HHHH:MM:SS:CC` em `time.Duration`, na mão (sem lib pronta).
**Entregável:** `func ParseIntervalo(s string) (time.Duration, error)` + testes.
**Regras de validação:**

| Formato | Campos | Faixas |
|---|---|---|
| `HH:MM:SS` | horas:min:seg | HH 00–99, MM 00–59, SS 00–59 |
| `HHHH:MM:SS:CC` | horas:min:seg:centésimos | HHHH 0000–9999, MM/SS 00–59, CC 00–99 |

**Conceitos novos:** `strings.Split`, `strconv.Atoi`, criação de `error` customizado (`errors.New`, `fmt.Errorf` com `%w`), **table-driven tests** com o pacote `testing`.

**Exemplo de teste table-driven** (padrão idiomático em Go, vale internalizar):
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

#### 2b — Parsing da expressão cron (estilo cron completo, escrito à mão)
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

### Módulo 3 — CLI
**Objetivo:** comandos `dsync task add|remove|ls|execute`.
**Entregável:** binário `dsync` funcional, operando só sobre o arquivo local (ainda sem daemon — isso é intencional, testamos a lógica de dados isolada primeiro).
**Conceitos novos:** parsing de argumentos (`os.Args` na mão, ou lib como `cobra`), formatação de tabela (`text/tabwriter`), detecção de tipo de ação (caminho existente vs `.go` vs comando).

**Sintaxe do `add` com os dois tipos de gatilho:**
```bash
dsync task add "echo bom dia" 00:00:10                 # gatilho: intervalo
dsync task add "backup.sh" --at "0 12 * * *"           # gatilho: horário fixo (cron)
```
O argumento posicional de intervalo e a flag `--at` são **mutuamente exclusivos** — se os dois vierem juntos, é erro de validação (mensagem clara, não silenciosamente ignora um dos dois).

**Decisão de design a discutir neste módulo:** usar `cobra` (mais "profissional", ensina padrão de CLIs Go do mundo real) ou implementar na mão com `os.Args` (mais didático pra entender o que uma lib como cobra abstrai). Podemos fazer os dois: primeiro na mão, depois migrar pra `cobra`.

---

### Módulo 4 — Execução das ações
**Objetivo:** implementar de fato o que cada tipo de task faz.
**Entregável:** `func Executar(t Task) error` cobrindo os 3 tipos.

| Tipo | Como executa |
|---|---|
| Comando | `exec.Command("sh", "-c", cmd)` no Unix / `exec.Command("cmd", "/C", cmd)` no Windows |
| Arquivo | `io.Copy` de origem pro destino |
| Código Go | `exec.Command("go", "build", ...)` na 1ª vez ou se hash mudou; depois só roda o binário em cache |

**Conceitos novos:** `os/exec`, captura de `stdout`/`stderr` (`cmd.Output()` vs `cmd.CombinedOutput()`), `io.Copy`, hashing de arquivo (`crypto/sha256`) pra detectar mudança no `.go`, `context.Context` pra timeout de execução.

---

### Módulo 5 — Scheduler (o coração concorrente)
**Objetivo:** processo que mantém timers rodando e dispara tasks nos intervalos certos.
**Entregável:** loop de agendamento rodando em background dentro do daemon.
**Conceitos novos — os mais "Go" do projeto:**
- **goroutines** (`go func() {...}()`) — uma por task ativa, ou um loop central com `time.Ticker`.
- **channels** — comunicação entre a goroutine do scheduler e o resto do programa (ex: canal de "task disparou", canal de "pare tudo").
- **`sync.Mutex`** — proteger a lista de tasks contra leitura/escrita simultânea (CLI e tray podem mexer ao mesmo tempo).
- **graceful shutdown** — capturar `SIGINT`/`SIGTERM` (`os/signal`) e desligar limpo, salvando estado antes de sair.
- Política de concorrência por task: se o intervalo disparar de novo enquanto a execução anterior ainda roda, **pula e loga aviso** (evita empilhar).

---

### Módulo 6 — Daemon + IPC
**Objetivo:** processo `dsyncd` separado, e a CLI conversando com ele.
**Entregável:** `dsync daemon start|stop|status` funcionando; `task add/remove/ls/execute` passam a falar com o daemon em vez de mexer direto no arquivo.
**Conceitos novos:** unix socket (`net.Listen("unix", ...)`) no Linux/macOS, named pipe no Windows, ou HTTP local como alternativa mais simples (`net/http` num `127.0.0.1:porta`), serialização de mensagens (JSON de novo), PID file, checagem de processo vivo.

**Nota de design:** por simplicidade multiplataforma, pode valer a pena começar com **HTTP local** (mais fácil, mesma API em qualquer SO) e só migrar pra unix socket/named pipe depois, se quiser otimizar.

---

### Módulo 7 — Logs
**Objetivo:** cada task grava histórico de execução.
**Entregável:** `~/.dsync/logs/<nome>.log` com timestamps, stdout/stderr, código de saída.
**Conceitos novos:** `log/slog` (stdlib, Go 1.21+), abrir arquivo em modo append (`os.OpenFile` com `os.O_APPEND`).

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
Isso importa pro DSync: por exemplo, o zero value de `time.Time` é `0001-01-01 00:00:00`, não "vazio" — por isso usamos `*time.Time` (ponteiro) quando queremos representar "ainda não aconteceu".

### 5.2 Structs e métodos
Struct é um tipo composto — agrupa campos relacionados (como o `Task`). Métodos são funções "presas" a um tipo, via **receiver**:
```go
func (t Task) EstaAtrasada() bool {
	return time.Now().After(t.ProximaExecucao)
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
No DSync isso aparece em dois lugares principais: `*time.Time` para "opcional", e receivers de método (`*Task`) para mutação.

### 5.4 Slices, arrays e maps
- **Array**: tamanho fixo, raramente usado direto.
- **Slice**: array dinâmico — o que você vai usar sempre. `[]Task` é uma slice de tasks.
- **Map**: dicionário chave→valor. Útil pra indexar tasks por nome: `map[string]Task`.

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
func LoadTasks() ([]Task, error) {
	dados, err := os.ReadFile(caminho)
	if err != nil {
		return nil, fmt.Errorf("erro lendo tasks: %w", err)
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

### 5.8 sync.Mutex
Quando múltiplas goroutines (ou o daemon + a CLI) podem ler/escrever a mesma coisa (a lista de tasks) ao mesmo tempo, é preciso proteger com um lock:
```go
type Store struct {
	mu    sync.Mutex
	tasks []Task
}

func (s *Store) Adicionar(t Task) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tasks = append(s.tasks, t)
}
```
`defer` executa a chamada no final da função, garantindo que o `Unlock` acontece mesmo se a função retornar mais cedo por erro.

### 5.9 defer, panic, recover
- `defer` adia uma chamada até o final da função (fecho de arquivo, unlock de mutex, etc.) — muito usado.
- `panic`/`recover` são para erros irrecuperáveis/bugs, não para fluxo normal de erro (isso é `error`, como na seção 5.5). Você provavelmente vai usar pouco disso no DSync, exceto talvez pra evitar que uma task travando derrube o daemon inteiro.

### 5.10 Pacotes e módulos
- Um **módulo** (`go.mod`) é o projeto inteiro; pode conter vários **pacotes** (pastas com `.go` dentro).
- `package main` é especial: é o ponto de entrada executável.
- Convenção sugerida pro DSync conforme crescer:
```
dsync/
├── go.mod
├── main.go              (CLI, package main)
├── cmd/dsyncd/main.go    (daemon, outro package main)
├── task/                 (package task: struct Task, parsing, persistência)
├── scheduler/             (package scheduler)
└── ipc/                   (package ipc)
```

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
Arquivo `*_test.go`, função `func TestAlgo(t *testing.T)`, roda com `go test ./...`. Padrão table-driven já mostrado na seção do Módulo 2 — vale adotar em todo o projeto, é o estilo idiomático de Go.

### 5.14 Cross-platform (Linux/macOS/Windows no mesmo binário)
- **Build tags / sufixo de arquivo**: `daemon_unix.go` vs `daemon_windows.go` — o compilador escolhe o arquivo certo pelo nome (`_windows.go`, `_linux.go`, `_darwin.go`) ou por diretiva `//go:build windows` no topo do arquivo.
- **`SysProcAttr`**: campo de `exec.Cmd` que varia por SO — é onde as diferenças de "processo desanexado" (Módulo 6) aparecem na prática.
- Isso vai ficar mais concreto quando chegarmos no daemon — por enquanto só fica de referência.

---

## 6. Referência rápida de pacotes stdlib usados no projeto

| Pacote | Pra que serve no DSync |
|---|---|
| `os` | home dir, args, arquivos, sinais |
| `encoding/json` | serializar/desserializar tasks |
| `path/filepath` | montar caminhos multiplataforma |
| `time` | duração, timestamps |
| `strings`, `strconv` | parsing do intervalo |
| `errors`, `fmt` | erros customizados |
| `testing` | testes |
| `os/exec` | rodar comando/compilar Go |
| `io` | cópia de arquivo |
| `crypto/sha256` | hash do `.go` pra saber se recompila |
| `context` | timeout de execução |
| `sync` | proteger dados compartilhados |
| `os/signal` | shutdown gracioso |
| `net` / `net/http` | IPC CLI↔daemon |
| `log/slog` | logging estruturado |
| `text/tabwriter` | saída em tabela do `task ls` |

---

## 7. Próximos passos imediatos

1. **Terminar Módulo 1** com a struct atualizada (`Gatilho` separado de `Task`): `SaveTasks`/`LoadTasks`.
2. Escrever um `main.go` mínimo que cria uma `Task` com `Gatilho{Tipo: GatilhoIntervalo, ...}` na mão, salva, recarrega e imprime — validação manual antes de qualquer CLI de verdade.
3. Seguir pro Módulo 2a (parsing de intervalo) com testes table-driven.
4. Módulo 2b (parser de cron) pode vir logo depois ou ser intercalado com o Módulo 3 (CLI) — como preferir.

---

*Documento vivo — atualizar conforme decisões forem fechadas e módulos avançarem.*