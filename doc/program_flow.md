# DockYard — Program Flow

Διαγράμματα ροής της εφαρμογής, βασισμένα στον πραγματικό κώδικα του backend (`/home/pc/DockYard/backend/`).

---

## 1. Αρχιτεκτονική Υψηλού Επιπέδου

```mermaid
flowchart TD
    Browser["Browser<br/>(HTML/CSS/Vanilla JS)"]

    subgraph Frontend["Frontend Container"]
        Nginx["Nginx<br/>(static files only)"]
    end

    subgraph Backend["Backend Container"]
        Gin["Gin Router<br/>:8081"]
        Auth["JWT Middleware"]
        Handlers["Handlers<br/>(auth, projects, docker, files, logs, terminal)"]
        Cleanup["Cleanup Worker<br/>(goroutine, 5min ticker)"]
        DockerSvc["Docker Service<br/>(Docker SDK for Go)"]
    end

    DB[("PostgreSQL<br/>users, projects")]
    Daemon["Docker Daemon<br/>(/var/run/docker.sock)"]

    Browser -->|"HTTP :5173"| Nginx
    Browser -->|"HTTP/WebSocket :8081<br/>(απευθείας, χωρίς proxy)"| Gin
    Gin --> Auth --> Handlers
    Handlers -->|GORM| DB
    Handlers --> DockerSvc
    Cleanup -->|raw SQL| DB
    Cleanup --> DockerSvc
    DockerSvc -->|Unix socket| Daemon
```

---

## 2. Εγγραφή και Σύνδεση Χρήστη (Register / Login)

```mermaid
sequenceDiagram
    actor U as Χρήστης
    participant F as Frontend (auth.js)
    participant B as Backend
    participant DB as PostgreSQL

    U->>F: Συμπλήρωση φόρμας εγγραφής
    F->>B: POST /api/register {email, password}
    B->>B: bcrypt.GenerateFromPassword (cost 12)
    B->>DB: INSERT INTO users
    B-->>F: 201 Created

    U->>F: Συμπλήρωση φόρμας σύνδεσης
    F->>B: POST /api/login {email, password}
    B->>DB: SELECT * FROM users WHERE email = ?
    B->>B: bcrypt.CompareHashAndPassword
    alt έγκυρα credentials
        B->>B: jwt.NewWithClaims(HS256, exp: 24h)
        B-->>F: 200 OK {token}
        F->>F: Αποθήκευση token (localStorage)
    else μη έγκυρα
        B-->>F: 401 Unauthorized
    end
```

---

## 3. Εκκίνηση Container (Launch Project)

Η πιο σύνθετη ροή του συστήματος. Χρησιμοποιεί atomic claim στη βάση δεδομένων για αποφυγή race condition (διπλή εκκίνηση από ταυτόχρονα αιτήματα), και εκτελεί την πραγματική εργασία ασύγχρονα σε goroutine.

```mermaid
sequenceDiagram
    actor U as Χρήστης
    participant F as Frontend
    participant H as Handler (LaunchContainer)
    participant DB as PostgreSQL
    participant G as goroutine
    participant D as Docker SDK
    participant Daemon as Docker Daemon

    U->>F: Κλικ "Launch"
    F->>H: POST /api/projects/:id/launch

    H->>DB: UPDATE projects SET status='pulling'<br/>WHERE id=? AND status IN ('stopped','error')
    alt RowsAffected = 0
        H-->>F: 409 Conflict (already running/pulling)
    else RowsAffected = 1 (claim επιτυχές)
        H->>DB: SELECT project
        H->>H: crypto.Decrypt(registry_password)
        H-->>F: 202 Accepted {status:"pulling"}

        H->>G: εκκίνηση goroutine (async)
        G->>G: validateMounts (sandbox σε FilesHostDir/projectID)
        G->>D: PullImage(image, registryUser, registryPassword)
        D->>Daemon: docker pull
        alt pull αποτυγχάνει
            G->>DB: UPDATE status='error'
        else pull επιτυχές
            G->>D: RunContainer(image, env, binds, port, terminalMode)
            D->>Daemon: ContainerCreate + ContainerStart
            Daemon-->>D: containerID
            D->>D: GetContainerPort (ContainerInspect)
            G->>DB: UPDATE status='running',<br/>container_id, port, started_at
        end
    end

    loop Polling από frontend
        F->>H: GET /api/projects/:id/status
        H->>DB: SELECT status, port
        H-->>F: {status, url}
    end
```

---

## 4. WebSocket Terminal

```mermaid
sequenceDiagram
    actor U as Χρήστης
    participant F as Frontend (project-details.js, xterm.js)
    participant H as Handler (TerminalWS)
    participant D as Docker SDK
    participant C as Container

    U->>F: Άνοιγμα terminal
    F->>H: WS /ws/projects/:id/terminal?token=JWT
    H->>H: Χειροκίνητη επικύρωση JWT (query param)
    H->>D: ContainerExecCreate (Tty: true, Cmd: /bin/sh)
    H->>D: ContainerExecAttach
    D->>C: exec session

    par Bidirectional bridge
        U->>F: Πληκτρολόγηση εντολής
        F->>H: WS message
        H->>C: write στο exec stream
    and
        C->>H: stdout/stderr
        H->>F: WS message
        F->>U: Εμφάνιση στο xterm.js
    end

    U->>F: Resize παραθύρου
    F->>H: WS resize event
    H->>D: ContainerExecResize
```

---

## 5. Αυτόματος Τερματισμός Αδρανών Containers (Cleanup Worker)

```mermaid
flowchart TD
    Start["main.go: go cleanup.StartWorker(db)"] --> Ticker["time.Ticker (5 λεπτά)"]
    Ticker -->|κάθε tick| Query["Raw SQL:<br/>status='running' AND<br/>last_accessed_at < NOW() - auto_stop_min<br/>AND auto_stop_min > 0"]
    Query --> Found{Βρέθηκαν<br/>αδρανή projects?}
    Found -->|Όχι| Ticker
    Found -->|Ναι| Loop["Για κάθε project"]
    Loop --> HasID{container_id<br/>≠ nil;}
    HasID -->|Όχι| Ticker
    HasID -->|Ναι| Stop["StopAndRemoveContainer(container_id)<br/>⚠ error αγνοείται"]
    Stop --> Update["UPDATE projects SET<br/>status='stopped', container_id=NULL, port=NULL"]
    Update --> Ticker
```

> **Σημείωση:** Ο worker δεν επαληθεύει την πραγματική κατάσταση του container μέσω `ContainerInspect` πριν την ενημέρωση της βάσης — βλ. Κεφάλαιο 8 για συζήτηση αυτού του ορίου σχεδιασμού.

---

## 6. Διαγραφή Χρήστη (Cascade Delete)

```mermaid
flowchart LR
    A["DELETE FROM users WHERE id = ?"] -->|ON DELETE CASCADE| B["Αυτόματη διαγραφή<br/>όλων των projects<br/>του χρήστη"]
    B --> C["⚠ Τα containers που<br/>τυχόν τρέχουν ΔΕΝ<br/>τερματίζονται αυτόματα"]
```
