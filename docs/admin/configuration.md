[Prev](/admin/building) - [Next](/admin/consumer)

# Configuration

Djinn CI uses a custom configuration language. The syntax of the configuration
files is detailed below using Extended Backus-Naur Form,

    digit  = "0" ... "9" .
    letter = "a" ... "z" | "A" ... "Z" | "_" | unicode_letter .

    identifier = letter { letter | digit } .

    float_literal  = digit "." { digit } .
    int_literal    = { digit } .
    number_literal = int_literal | float_literal .

    duration_unit    = "s" | "m" | "h" .
    duration_literal = number_literal { number_literal | duration_unit } .

    size_unit    = "B" | "KB" | "MB" | "GB" | "TB" .
    size_literal = int_literal size_unit .

    string_literal = `"` { letter } `"` .

    bool_literal = "true" | "false" .

    literal = bool_literal | string_literal | number_literal | duration_literal | size_literal .

    block   = "{" [ parameter ";" ] "}" .
    array   = "[" [ operand "," ] "]" .
    operand = literal | array | block .

    parameter = identifier [ identifier ] operand .

    file = { parameter ";" } .

All of the configuration files used, with the exception of `driver.conf`, share
the following configuration parameters,

* **`pidfile`** `string`

Defines the file to which the PID should be written when the process starts.

    pidfile "/var/run/djinn/server.pid"

* **`log`** `identifier` `string`

Configure the logging and level for the process.

    log info "/var/log/djinn/server.log"

*`level`* must be one of either `debug`, `info`, `warn`, or `error`.

The `include` keyword can be used to include other configuration files. This
can be useful when multiple configuration file share the same configuration
parameters, such as SMTP connections, for example,

    include "/etc/djinn/smtp.cfg"

this can also be given an array of paths to include,

    include [
        "/etc/djinn/crypto.cfg",
        "/etc/djinn/smtp.cfg",
    ]
