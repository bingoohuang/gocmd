# gocmd package

A simple package to execute shell commands on linux, darwin and windows.

## Installation

`go get -u github.com/bingoohuang/gocmd@latest`

## Usage

```go
c := gocmd.New("echo hello")

err := c.Run(context.TODO())

fmt.Println(c.Stdout())
fmt.Println(c.Stderr())


// execute shell file with arguments

shellCmd, err := shellquote.Quote("a.sh", "arg1", "args")
c2 := gocmd.New(shellCmd)
```

## Configure the command

To configure the command an option function can be passed which receives the
command object as an argument passed by reference.

Default option functions:

```
gocmd.WithBaseCommand(*exec.Cmd)
gocmd.WithStdStreams()
gocmd.WithStdout(...io.Writers)
gocmd.WithStderr(...io.Writers)
gocmd.WithTimeout(time.Duration)
gocmd.WithWorkingDir(string)
gocmd.WithEnv(gocmd.EnvVars)
```

### Example

```go
c := gocmd.New("echo hello", 
	gocmd.WithStdStreams(), 
	gocmd.WithWorkingDir("/tmp/test"),
	gocmd.WithStdout(linestream.New(func(line string) {
	    fmt.Println(line)
    })))
c.Run(context.TODO())
```

## resources

1. [Go Exec 僵尸与孤儿进程](https://github.com/WilburXu/blog/blob/master/Golang/Go%20Exec%20%E5%83%B5%E5%B0%B8%E4%B8%8E%E5%AD%A4%E5%84%BF%E8%BF%9B%E7%A8%8B.md)
2. [commander-cli/cmd](https://github.com/commander-cli/cmd) A simple package to execute shell commands on linux, darwin and windows.
3. [o os/exec 简明教程](https://colobu.com/2020/12/27/go-with-os-exec/)
4. [ionrock/procs](https://github.com/ionrock/procs) is a library to make working with command line applications a little nicer.
