// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build ignore

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
)

func main() {
	flag.Parse()

	args, target := flag.Args(), "peg"
	if len(args) > 0 {
		target = args[0]
	}

	switch target {
	case "peg":
		peg()
	case "clean":
		clean()
	case "test":
		test()
	case "bench":
		bench()
	case "help":
		fmt.Println("go run build.go [target]")
		fmt.Println(" peg - build peg from scratch")
		fmt.Println(" clean - clean up")
		fmt.Println(" test - run full test")
		fmt.Println(" bench - run benchmark")
	}
}

var processed = make(map[string]bool)

func done(file string, deps ...interface{}) bool {
	fini := true
	file = filepath.FromSlash(file)
	info, err := os.Stat(file)
	if err != nil {
		fini = false
	}
	for _, dep := range deps {
		switch dep := dep.(type) {
		case string:
			if info == nil {
				fini = false
				break
			}
			dep = filepath.FromSlash(dep)
			fileInfo, err := os.Stat(dep)
			if err != nil {
				panic(err)
			}

			if fileInfo.ModTime().After(info.ModTime()) {
				fini = false
			}
		case func() bool:
			name := runtime.FuncForPC(reflect.ValueOf(dep).Pointer()).Name()
			if result, ok := processed[name]; ok {
				fini = fini && result
				fmt.Printf("%s is done\n", name)
				break
			}
			result := dep()
			fini = fini && result
			fmt.Printf("%s\n", name)
			processed[name] = result
		}
	}

	return fini
}

func chdir(dir string) string {
	dir = filepath.FromSlash(dir)
	working, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	err = os.Chdir(dir)
	if err != nil {
		panic(err)
	}
	fmt.Printf("cd %s\n", dir)
	return working
}

func command(name, inputFile, outputFile string, arg ...string) {
	name = filepath.FromSlash(name)
	inputFile = filepath.FromSlash(inputFile)
	outputFile = filepath.FromSlash(outputFile)
	fmt.Print(name)
	for _, a := range arg {
		fmt.Printf(" %s", a)
	}

	cmd := exec.Command(name, arg...)

	if inputFile != "" {
		fmt.Printf(" < %s", inputFile)
		input, err := ioutil.ReadFile(inputFile)
		if err != nil {
			panic(err)
		}
		writer, err := cmd.StdinPipe()
		if err != nil {
			panic(err)
		}
		go func() {
			defer writer.Close()
			_, err := writer.Write([]byte(input))
			if err != nil {
				panic(err)
			}
		}()
	}

	if outputFile != "" {
		fmt.Printf(" > %s\n", outputFile)
		output, err := cmd.Output()
		if err != nil {
			panic(err)
		}
		err = ioutil.WriteFile(outputFile, output, 0600)
		if err != nil {
			panic(err)
		}
	} else {
		output, err := cmd.CombinedOutput()
		fmt.Printf("\n%s", string(output))
		if err != nil {
			panic(err)
		}
	}
}

func delete(file string) {
	file = filepath.FromSlash(file)
	fmt.Printf("rm -f %s\n", file)
	os.Remove(file)
}

func deleteFilesWithSuffix(suffix string) {
	files, err := ioutil.ReadDir(".")
	if err != nil {
		panic(err)
	}
	for _, file := range files {
		if strings.HasSuffix(file.Name(), suffix) {
			delete(file.Name())
		}
	}
}

func bootstrap() bool {
	if done("bootstrap/bootstrap", "bootstrap/main.go", "tree/peg.go") {
		return true
	}

	wd := chdir("bootstrap")
	defer chdir(wd)

	command("go", "", "", "build")

	return false
}

func peg0() bool {
	if done("cmd/peg-bootstrap/peg0", "cmd/peg-bootstrap/main.go", bootstrap) {
		return true
	}

	wd := chdir("cmd/peg-bootstrap/")
	defer chdir(wd)

	deleteFilesWithSuffix(".peg.go")
	command("../../bootstrap/bootstrap", "", "")
	command("go", "", "", "build", "-tags", "bootstrap", "-o", "peg0")

	return false
}

func peg1() bool {
	if done("cmd/peg-bootstrap/peg1", peg0, "cmd/peg-bootstrap/bootstrap.peg") {
		return true
	}

	wd := chdir("cmd/peg-bootstrap/")
	defer chdir(wd)

	deleteFilesWithSuffix(".peg.go")
	command("./peg0", "bootstrap.peg", "peg1.peg.go")
	command("go", "", "", "build", "-tags", "bootstrap", "-o", "peg1")

	return false
}

func peg2() bool {
	if done("cmd/peg-bootstrap/peg2", peg1, "cmd/peg-bootstrap/peg.bootstrap.peg") {
		return true
	}

	wd := chdir("cmd/peg-bootstrap/")
	defer chdir(wd)

	deleteFilesWithSuffix(".peg.go")
	command("./peg1", "peg.bootstrap.peg", "peg2.peg.go")
	command("go", "", "", "build", "-tags", "bootstrap", "-o", "peg2")

	return false
}

func peg3() bool {
	if done("cmd/peg-bootstrap/peg3", peg2, "peg.peg") {
		return true
	}

	wd := chdir("cmd/peg-bootstrap/")
	defer chdir(wd)

	deleteFilesWithSuffix(".peg.go")
	command("./peg2", "../../peg.peg", "peg3.peg.go")
	command("go", "", "", "build", "-tags", "bootstrap", "-o", "peg3")

	return false
}

func peg_bootstrap() bool {
	if done("cmd/peg-bootstrap/peg-bootstrap", peg3) {
		return true
	}

	wd := chdir("cmd/peg-bootstrap/")
	defer chdir(wd)

	deleteFilesWithSuffix(".peg.go")
	command("./peg3", "../../peg.peg", "peg-bootstrap.peg.go")
	command("go", "", "", "build", "-tags", "bootstrap", "-o", "peg-bootstrap")

	return false
}

func peg_peg_go() bool {
	if done("peg.peg.go", peg_bootstrap) {
		return true
	}

	command("cmd/peg-bootstrap/peg-bootstrap", "peg.peg", "peg.peg.go")
	command("go", "", "", "build")
	command("./peg", "", "", "-inline", "-switch", "peg.peg")

	return false
}

func peg() bool {
	if done("peg", peg_peg_go, "main.go") {
		return true
	}

	command("go", "", "", "build")

	return false
}

func clean() bool {
	delete("bootstrap/bootstrap")

	delete("grammars/c/c.peg.go")
	delete("grammars/calculator/calculator.peg.go")
	delete("grammars/fexl/fexl.peg.go")
	delete("grammars/java/java_1_7.peg.go")
	delete("grammars/long_test/long.peg.go")

	wd := chdir("cmd/peg-bootstrap/")
	defer chdir(wd)

	deleteFilesWithSuffix(".peg.go")
	delete("peg0")
	delete("peg1")
	delete("peg2")
	delete("peg3")
	delete("peg-bootstrap")

	return false
}

func grammars_c() bool {
	if done("grammars/c/c.peg.go", peg, "grammars/c/c.peg") {
		return true
	}

	wd := chdir("grammars/c/")
	defer chdir(wd)

	command("../../peg", "", "", "-switch", "-inline", "c.peg")

	return false
}

func grammars_calculator() bool {
	if done("grammars/calculator/calculator.peg.go", peg, "grammars/calculator/calculator.peg") {
		return true
	}

	wd := chdir("grammars/calculator/")
	defer chdir(wd)

	command("../../peg", "", "", "-switch", "-inline", "calculator.peg")

	return false
}

func grammars_fexl() bool {
	if done("grammars/fexl/fexl.peg.go", peg, "grammars/fexl/fexl.peg") {
		return true
	}

	wd := chdir("grammars/fexl/")
	defer chdir(wd)

	command("../../peg", "", "", "-switch", "-inline", "fexl.peg")

	return false
}

func grammars_java() bool {
	if done("grammars/java/java_1_7.peg.go", peg, "grammars/java/java_1_7.peg") {
		return true
	}

	wd := chdir("grammars/java/")
	defer chdir(wd)

	command("../../peg", "", "", "-switch", "-inline", "java_1_7.peg")

	return false
}

func grammars_long_test() bool {
	if done("grammars/long_test/long.peg.go", peg, "grammars/long_test/long.peg") {
		return true
	}

	wd := chdir("grammars/long_test/")
	defer chdir(wd)

	command("../../peg", "", "", "-switch", "-inline", "long.peg")

	return false
}

func test() bool {
	if done("", grammars_c, grammars_calculator, grammars_fexl, grammars_java, grammars_long_test) {
		return true
	}

	command("go", "", "", "test", "-short", "-tags", "grammars", "./...")

	return false
}

func bench() bool {
	peg()

	command("go", "", "", "test", "-benchmem", "-bench", ".")

	return false
}
