package mghash

import "github.com/bobg/mghash"

type protoCmd struct {
	name      string
	goOut     string
	dirs      []string
	otherArgs []string
}

// Proto produces a Rule for compiling protocol buffers to Go.
func Proto(sources, targets []string, options ...ProtoOpt) mghash.Rule {
	cmd := ProtoCmd{
		name:  "protoc",
		goOut: "--go_out=.",
		dirs:  []string{"."},
	}

	for _, opt := range options {
		opt(&cmd)
	}

	command := []string{cmd.name}
	if cmd.goOut != "" {
		command = append(command, cmd.goOut)
	}
	for _, dir := range cmd.dirs {
		command = append(command, "-I"+dir)
	}
	for _, arg := range cmd.otherArgs {
		command = append(command, arg)
	}
	command = append(command, sources...)

	return JRule{
		Sources: sources,
		Targets: targets,
		Command: command,
	}
}

type ProtoOpt func(*protoCmd)

func Protoc(name string) ProtoOpt {
	return func(cmdptr *protoCmd) {
		cmdptr.name = name
	}
}

func ProtoDirs(dirs...string) ProtoOpt {
	return func(cmdptr *protoCmd) {
		cmd.dirs = append(cmd.dirs, dirs...)
	}
}

func ProtocArgs(args ...string) ProtoOpt {
	return func(cmdptr *protoCmd) {
		protoCmd.otherArgs = append(protoCmd.otherArgs, args...)
	}
}
