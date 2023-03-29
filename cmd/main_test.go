package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"testing"
)

func TestMain(m *testing.M) {
	appName := "gwiot"
	fmt.Println("-> Building...")
	// checking if Microsoft Windows OS
	// if so, we append .exe to make it excutable by Go
	if runtime.GOOS == "windows" {
		appName += ".exe"
	}
	// We run a command to build the code
	// we are using appName string as tthe name of the excutable
	build := exec.Command("go", "build", "-o", appName)
	if err := build.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error building %s: %s", appName, err)
		os.Exit(1)
	}
	fmt.Println("-> Running...")
	// Running the
	result := m.Run()
	fmt.Println("-> Getting done...")
	os.Remove(appName)
	os.Exit(result)
}
