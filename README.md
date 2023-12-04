Glug
====

![glug glug glug](https://media.giphy.com/media/6QAMESWJ82WAg/200.gif)

Self-rotating file logger for `runit`

## Usage

Place in `$SV_DIR/$SERVICE/log/run`:

```
#!/bin/sh

exec glug /var/log/service.log
```

Run `glug help` for complete CLI usage:

```
% ./glug help
Self-rotating file logger for runit

CALL TYPE
Usage:
  glug LOGFILE [flags]
  glug [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  rotate      Perform rotation upon the specified log file

Flags:
      --count int              Number of rotated log-files to retain (default 4)
  -h, --help                   help for glug
      --max-age duration       Maximum age for the output log-file (default 168h0m0s)
      --max-size memory.Size   Maximum byte-size of the output log-file (default 32.0 MiB)
      --min-size memory.Size   Block rotation of small log-files by age until they reach a minimum size threshold (default 512.0 KiB)
      --mode int               Mode bits for log-file creation. Octal values are supported with a leading 0 (default 0644)
      --pattern string         strftime format string for rotated file name suffixes (default "%Y-%m-%dT%H%M%S")
      --rotate                 Enable log rotation (default true)

Use "glug [command] --help" for more information about a command.
```
