# Examen de Programmation Go — Session 2026

- **Durée** : 2 heures
- **Documents** : autorisés
- **IA** : autorisée (ChatGPT, Claude, Copilot, etc.)
- **Langage** : Go (uniquement la bibliothèque standard, pas de dépendances externes)

---

## Sujet — `taskrunner` : Orchestrateur de tâches concurrentes

Vous devez développer un programme en ligne de commande qui lit un fichier de tâches, les exécute en parallèle avec un pool de workers, gère les timeouts et les retries, puis produit un rapport JSON.

---

## 1. Format du fichier d'entrée

Le programme lit un fichier `tasks.json` contenant une liste de tâches :

```json
{
  "tasks": [
    {
      "id": "t1",
      "type": "print",
      "params": { "message": "hello" },
      "timeout": "2s",
      "retries": 0
    },
    {
      "id": "t2",
      "type": "download",
      "params": { "url": "https://example.com/data.json", "dest": "/tmp/data.json" },
      "timeout": "5s",
      "retries": 2
    },
    {
      "id": "t3",
      "type": "calc",
      "params": { "value": 42 },
      "timeout": "1s",
      "retries": 1
    }
  ]
}
```

Chaque tâche possède :
- `id` : identifiant unique (chaîne)
- `type` : type de la tâche (`"print"`, `"download"`, `"calc"`, `"fake"`, etc.)
- `params` : paramètres spécifiques au type de tâche (objet JSON)
- `timeout` : durée maximale d'exécution (format Go : `"2s"`, `"500ms"`, etc.)
- `retries` : nombre de tentatives en cas d'échec ou timeout (en plus de la première tentative)

Le chargement du fichier doit utiliser un `switch` sur le champ `type` pour instancier la bonne implémentation de `Task`.

**Important** : les tâches sont implémentées en pur Go. Aucune commande shell ne doit être exécutée (`exec.Command`, `os/exec`, etc.).

---

## 2. Comportement attendu

### Lancement

```bash
taskrunner -file tasks.json -workers 3
```

- `-file` : chemin vers le fichier JSON (obligatoire)
- `-workers` : nombre de workers simultanés (défaut : 3)

### Exécution

- Le type de chaque tâche détermine comment elle est exécutée.
- Chaque implémentation de `Task` effectue une action en pur Go (affichage, téléchargement, calcul, etc.).
- Chaque tâche doit respecter son `timeout` via `context.WithTimeout`.
- Si une tâche dépasse le timeout et `retries > 0`, elle est relancée.
- Si une tâche retourne une erreur et `retries > 0`, elle est relancée.
- Le nombre de workers simultanés est limité par le flag `-workers`.

**Interdit** : `os/exec`, `exec.Command`, ou tout appel à un shell.

### Rapport de sortie

À la fin de l'exécution, le programme affiche sur **stdout** un rapport JSON. La durée doit être formatée avec `time.Duration.String()`.

```json
{
  "results": [
    { "id": "t1", "status": "success", "duration": "12ms", "attempts": 1 },
    { "id": "t2", "status": "timeout", "duration": "3.001s", "attempts": 3 },
    { "id": "t3", "status": "failed", "duration": "45ms", "attempts": 2 }
  ]
}
```

Valeurs possibles pour `status` : `"success"`, `"failed"`, `"timeout"`.

---

## 3. Contraintes obligatoires

### 3.1 Interface Task

Vous devez définir une **interface `Task`** avec au moins les méthodes :
- `ID() string`
- `Execute(ctx context.Context) error`

Vous devez fournir **plusieurs implémentations** de cette interface, par exemple :
- `PrintTask` : affiche un message
- `DownloadTask` : télécharge un fichier depuis une URL
- `CalcTask` : effectue un calcul simple
- `FakeTask` : simule une tâche pour les tests

Chaque implémentation doit être en pur Go. L'utilisation de `os/exec` ou de commandes shell est interdite.

L'implémentation `FakeTask` doit être constructible de manière standardisée, par exemple avec un type énuméré pour choisir le comportement :

```go
type FakeTaskBehavior int

const (
    BehaviorSuccess FakeTaskBehavior = iota
    BehaviorFail
    BehaviorTimeout
)

func NewFakeTask(id string, behavior FakeTaskBehavior, delay time.Duration) *FakeTask
```

### 3.2 Type d'erreur custom

Vous devez définir un type d'erreur :

```go
type TaskError struct {
    Code   int
    TaskID string
    Err    error
}
```

Ce type doit implémenter les méthodes `Error() string` et `Unwrap() error`.

Toutes les erreurs retournées par le programme doivent être wrappées avec ce type ou avec `fmt.Errorf("...: %w", err)`.

### 3.3 Validation des workers

Le flag `-workers` doit être validé par une fonction :

```go
func ValidateWorkers(n int) (int, error)
```

- Si `n < 1` ou `n > 100` → retourne `3, error`
- Sinon → retourne `n, nil`

### 3.4 Rapport via io.WriterTo

Le type `Report` doit implémenter l'interface `io.WriterTo` :

```go
func (r Report) WriteTo(w io.Writer) (n int64, err error)
```

Le rapport JSON doit être sérialisé via cette méthode.

### 3.5 Fonction Orchestrate

Votre programme doit définir une fonction principale **Orchestrate** avec la signature suivante :

```go
func Orchestrate(ctx context.Context, tasks []task.Task, workers int) (report.Report, error)
```

`main()` ne doit contenir que le parsing des flags et l'appel à `Orchestrate`.

### 3.6 Functional options

L'orchestrateur doit être configurable via le pattern **functional options**. Vous devez définir au moins :

```go
type Option func(*OrchestratorConfig)

func WithWorkers(n int) Option
func WithVerbose(v bool) Option
```

### 3.7 Mode verbose

Le programme doit accepter un flag `-verbose` qui affiche en temps réel le statut de chaque tâche sur **stderr** (début, succès, échec, timeout).

### 3.8 Graceful shutdown

Le programme doit gérer le signal `SIGINT` (Ctrl+C) de manière gracieuse :
- les tâches en cours sont annulées via le `context`
- le rapport partiel est quand même affiché sur stdout
- le fichier `METRICS.md` est généré

---

## 4. Tests unitaires

Votre projet doit contenir des **tests unitaires pertinents**.

---

## 5. Qualité du code

Votre code doit au moins passer les vérifications suivantes sans erreur :

```bash
go vet ./...
gofmt -d .
```

Aucune sortie = OK.

---

## 6. Livrables

### 6.1 Structure attendue

```
taskrunner/
├── go.mod
├── Makefile
├── tasks.json (fichier de test fourni par vous, avec différents types de tâches)
├── cmd
│   └── taskrunner/
│       └── main.go
├── ...
└── ...
```

### 6.2 Makefile

Le Makefile doit au moins contenir les cibles suivantes :

```bash
make build    # compile le binaire
make test     # lance les tests
make run      # compile et exécute avec tasks.json
make lint     # lance go vet et gofmt
```

### 6.3 Dépôt Git

Vous devez initialiser un dépôt git. Vous devrez commit et push au fur et à mesure de votre travail. 

### 6.4 Fichier METRICS.md

Votre programme doit générer automatiquement un fichier `METRICS.md`. Vous devez implémenter une fonction dédiée avec la signature suivante :

```go
func WriteMetrics(results []report.TaskResult) string
```

Cette fonction retourne le contenu Markdown du fichier. Elle doit contenir au minimum :
- le nombre de goroutines actives à la fin (`runtime.NumGoroutine()`)
- le nombre total de tâches exécutées
- le nombre de tâches réussies, en échec et en timeout

Exemple de contenu généré :

```markdown
# Métriques d'exécution

- Goroutines actives à la fin : 1
- Tâches exécutées : 3
- Tâches réussies : 1
- Tâches en échec : 1
- Tâches en timeout : 1
```

---

## 7. Barème indicatif

| Critère | Points  |
|---|---------|
| Fonctionnement correct | 25      |
| Contraintes obligatoires | 25      |
| Tests unitaires | 25      |
| Qualité du code | 25      |
| **Total** | **100** |

---

## 8. Rappel

L'utilisation de l'IA est autorisée. Cependant, vous êtes responsable de chaque ligne de code que vous rendez.
Votre code sera évalué à la fois sur son fonctionnement et sur sa qualité.

**Bon courage.**
