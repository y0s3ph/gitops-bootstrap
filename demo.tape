# gostrap — From zero to GitOps in one command
#
# Prerequisites:
#   go install github.com/charmbracelet/vhs@latest
#   go build -o gostrap ./cmd/gostrap/
#   export PATH="$PWD:$PATH"
#
# Generate:
#   vhs demo.tape

Output demo.gif

Set Shell "bash"
Set FontSize 16
Set Width 900
Set Height 550
Set TypingSpeed 60ms
Set Theme { "name": "gostrap", "black": "#1e1e2e", "red": "#f38ba8", "green": "#a6e3a1", "yellow": "#f9e2af", "blue": "#89b4fa", "magenta": "#cba6f7", "cyan": "#94e2d5", "white": "#cdd6f4", "brightBlack": "#585b70", "brightRed": "#f38ba8", "brightGreen": "#a6e3a1", "brightYellow": "#f9e2af", "brightBlue": "#89b4fa", "brightMagenta": "#cba6f7", "brightCyan": "#94e2d5", "brightWhite": "#cdd6f4", "background": "#1e1e2e", "foreground": "#cdd6f4", "selection": "#45475a", "cursor": "#f5e0dc" }

# --- gostrap init ---

Type "gostrap init"
Enter
Sleep 1.5s

# Form 1: GitOps controller (ArgoCD is pre-selected)
Enter
Sleep 500ms

# Controller version (accept default 2.13.1)
Enter
Sleep 1s

# Form 2: Secrets management (Sealed Secrets is pre-selected)
Enter
Sleep 1s

# Form 3: Manifest format (Kustomize is pre-selected)
Enter
Sleep 500ms

# Environments (accept default dev,staging,production)
Enter
Sleep 500ms

# Scaffold example app? (Yes is default)
Enter
Sleep 1s

# Form 4: Target repo path
Type "./my-gitops-repo"
Enter
Sleep 500ms

# Cluster context (leave empty — skip install)
Enter
Sleep 3s

# --- gostrap add-app ---

Type "gostrap add-app payments-api 3000 --repo-path ./my-gitops-repo"
Enter
Sleep 2s

# --- gostrap validate ---

Type "gostrap validate ./my-gitops-repo"
Enter
Sleep 2s

# --- gostrap diff ---

Type "gostrap diff dev production --app payments-api --repo-path ./my-gitops-repo"
Enter
Sleep 3s

# Pause at the end
Sleep 2s
