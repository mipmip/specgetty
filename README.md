# specgetty

![specgetty demo](demo.gif)

Do you work with multiple projects that use OpenSpec for managing specifications?

Have you ever lost track of which OpenSpec projects exist on your local machine
or what state they're in?

`spg` (specgetty) is a text-mode UI tool to find and report the status of
OpenSpec projects on your local machine.

## Source-mode installation

```bash
go install github.com/mipmip/specgetty@master
```

## Configuration

Copy [config.yml](src/config.yml) to `~/.config/specgetty/config.yml` and edit to your needs.

The config path follows the XDG Base Directory Specification. If `$XDG_CONFIG_HOME` is set, the config is read from `$XDG_CONFIG_HOME/specgetty/config.yml`.

## Running

```bash
spg [ <directories...> ]
```

If one/more directories are specified as `<directories>`, then this will override the
`scandirs.include` from your config file.

## UI

Simple key navigation in the UI as follows:

| Key                        | Action                                           |
| -------------------------- | ------------------------------------------------ |
| `j`/`k` or `<up>`/`<down>` | Navigation inside repositories or diff views     |
| `<tab>`                    | switch focus between repositories and diff views |
| `<enter>`                  | Open terminal in selected repo directory         |
| `s`                        | Rescan directories                               |
| `q` / `ctrl-C`             | quit                                             |

## Development

```bash
make lint
```

### Generating the preview GIF

The preview GIF is generated using [VHS](https://github.com/charmbracelet/vhs):

```bash
vhs demo.tape
```
