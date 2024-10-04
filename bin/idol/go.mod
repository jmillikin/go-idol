module go.idol-lang.org/bin/idol

go 1.23.1

require (
	github.com/spf13/cobra v1.8.1
	github.com/spf13/pflag v1.0.5
	github.com/tetratelabs/wazero v1.8.0
	go.idol-lang.org/idol v0.0.0
)

require github.com/inconshreveable/mousetrap v1.1.0 // indirect

replace go.idol-lang.org/idol v0.0.0 => ../../idol
