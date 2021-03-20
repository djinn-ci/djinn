[Prev](/admin/building) - [Next](/admin/curator)

# Configuration

Djinn CI uses it's own directive based configuration language for each of the
components. The syntax of the configuration files is detailed below using
Extended Backus-Naur Form:

    digit  = "0" ... "9" .
    letter = "a" ... "z" | "A" ... "Z" | "-" | "_" .

    identifier = letter { letter | digit } .

    bool_literal   = "true" | "false" .
    number_literal = [ digit ] .
    string_literal = `"` [ unicode_char ] `"` .

    literal = bool_literal | string_literal | number_literal .

    array_directive = identifier "[" literal [ "," ] "]" .
    block_directive = identifier [ identifier ] "{" { directive } "}" .
    value_directive = identifier [ identifier ] literal .

    directive = array_directive | block_directive | value_directive .

All of the configuration files used do share the following configuration
directives,

* **`pidfile`** `string`

Defines the file to which the PID should be written when the process starts.

    pidfile "/var/run/djinn/server.pid"

* **`log`** `identifier` `string`

Configure the logging and level for the process.

    log info "/var/log/djinn/server.log"

*`level`* must be one of either `debug`, `info`, `warn`, or `error`.
