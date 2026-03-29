set shell := ["bash", "-u", "-c"]

# export environment variables
export GOBIN := `echo $PWD/.bin`
export GOTOOLCHAIN := 'go1.26.1'
export scripts := ".github/workflows/scripts"
export race := if env_var_or_default("CGO_ENABLED", "1") == "1" { "-race" } else { "" }

# print available commands
[private]
default:
    @just --list

# run specific unit test
[group('testing')]
[no-cd]
test unit:
    go test -v -count=1 {{race}} -run {{unit}} 2>/dev/null

# run tests across the source tree
[group('testing')]
tests:
    go test -count=1 {{race}} ./... 2>/dev/null

# vet the source tree
[group('testing')]
vet:
    go vet ./...

# lint the source tree
[group('testing')]
lint: vet
    $GOBIN/golangci-lint --config $scripts/golangci.yaml run

# tidy up Go modules
[group('build')]
tidy:
    go mod tidy

# show host system information
[group('setup')]
@sysinfo:
    echo "{{os()/arch()}} {{num_cpus()}}c"

# locally install build tools
[group('setup')]
init:
    go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.11.4
