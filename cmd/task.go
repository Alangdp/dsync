/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/spf13/cobra"
)

var taskCmd = &cobra.Command{
	Use:   "task",
	Short: "Scheduled Task Manager",
	Long:  `Scheduled Task Manager: Manages all scheduled tasks registered in DSync.`,
}

func init() {
	rootCmd.AddCommand(taskCmd)

	// Adiciona o parâmetro tag foo no help
	// Exemplo: "--foo string   A help for foo"
	taskCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Adiciona o parâmetro booleano toggle/t no help
	// Exemplo: "-t, --toggle       Help message for toggle"
	taskCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
