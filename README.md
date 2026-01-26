# Clank

Minimal task/project management with a kanban-style web UI and REST API.

## Features

- Kanban board with drag-and-drop (Backlog, To Do, In Progress, Done)
- Projects to organize tasks
- Task hierarchy with subtasks
- Markdown specs with fullscreen editor and preview
- Mobile-friendly with tab navigation
- REST API for CLI and automation
- Single binary with embedded web UI

## Building

Requires [Nix](https://nixos.org/) with flakes enabled:

```bash
nix build
./result/bin/clank serve --port 8080
```

Or with Go directly:

```bash
go build ./cmd/clank
./clank serve --port 8080
```

## Usage

### Web UI

Open `http://localhost:8080` in your browser.

### CLI

The same binary includes a CLI client. Set the API URL:

```bash
export CLANK_API=http://localhost:8080
```

Commands:

```bash
# Projects
clank project list
clank project new "My Project"
clank project show <id>
clank project delete <id>

# Tasks
clank add "Task title" -p <project-id>
clank list [-p <project-id>] [-s <status>]
clank show <id>
clank edit <id> --title "New title" --status todo
clank move <id> -s in_progress --position 1.5
clank delete <id>
```

### API

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | /api/projects | List projects |
| POST | /api/projects | Create project |
| GET | /api/projects/:id | Get project |
| DELETE | /api/projects/:id | Delete project |
| GET | /api/tasks | List tasks |
| POST | /api/tasks | Create task |
| GET | /api/tasks/:id | Get task |
| PATCH | /api/tasks/:id | Update task |
| DELETE | /api/tasks/:id | Delete task |
| POST | /api/tasks/:id/move | Move task (status/position) |

## NixOS Deployment

A NixOS module is included for deployment:

```nix
{
  imports = [ inputs.clank.nixosModules.default ];

  services.clank = {
    enable = true;
    port = 8080;
    dataDir = "/var/lib/clank";
  };
}
```

## CLI with Pre-configured API URL

If you're running the Clank server on a remote machine (e.g., on your Tailnet), you can create a wrapped CLI package with a pre-configured API URL:

```nix
# In your flake.nix inputs
inputs.clank.url = "github:goldie/clank";

# In your NixOS/nix-darwin configuration
{ inputs, pkgs, ... }:
{
  environment.systemPackages = [
    (inputs.clank.lib.${pkgs.system}.mkCli {
      apiUrl = "http://your-machine.tailnet:8080";
    })
  ];
}
```

This creates a `clank` binary that automatically connects to the specified API URL without needing to set `CLANK_API`.

## License

MIT
