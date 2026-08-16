/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/Alangdp/dsync.git/internal/app"
	tasks "github.com/Alangdp/dsync.git/internal/task"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	if _, err := app.GetConfig(); err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	taskManager, err := tasks.GetInstance()
	if err != nil {
		slog.Error("failed to load tasks", "error", err)
	}

	fmt.Print(taskManager.List())

	// cmd.Execute()
}
