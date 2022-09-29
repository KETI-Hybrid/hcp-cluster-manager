package util

import (
	"encoding/json"
	"io"
	"log"
	"os/exec"
	"strings"
)

func CombinedOutput2(cmd *exec.Cmd) ([]byte, []byte) {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatal(err)
	}

	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	errb, _ := io.ReadAll(stderr)
	outb, _ := io.ReadAll(stdout)
	cmd.Wait()
	return errb, outb
}

func GetOutput(cmd *exec.Cmd) ([]byte, error) {
	errb, outb := CombinedOutput2(cmd)
	output := Output{
		Stderr: errb,
		Stdout: outb,
	}
	data, err := json.Marshal(output)
	return data, err
}

func GetOutputReplaceStr(cmd *exec.Cmd, old string, new string) ([]byte, error) {
	errb, outb := CombinedOutput2(cmd)
	output := Output{
		Stderr: errb,
		Stdout: outb,
	}

	// replace old string to new string
	temp := strings.ReplaceAll(string(output.Stderr), old, new)
	//regex, err := regexp.Compile("\n\n")
	// if err != nil {
	// 	return nil, err
	// }

	// remove a blank line from a multi-line string
	temp = strings.TrimSuffix(temp, "\n\n")
	output.Stderr = []byte(temp)
	data, err := json.Marshal(output)
	return data, err
}
