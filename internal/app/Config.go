package app

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"

	"github.com/Alangdp/dsync.git/internal/filesystem"
)

// Configuração do DSync
type Config struct {
	TasksPath string
}

func GetEmptyConfig() Config {
	// Caminho do app
	configPath := GetDsyncPath()

	return Config{
		TasksPath: filepath.Join(configPath, "tasks.json"),
	}
}

func SaveConfiguration(config Config) {
	// Caminho do app
	configPath := GetDsyncPath()

	jsonData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		slog.Error("Error on save configuration (Marshal)")
		return
	}

	configFile := filepath.Join(configPath, "config.json")

	err = os.WriteFile(configFile, jsonData, 0644)
	if err != nil {
		slog.Error("Error on save configuration (Save file)")
		return
	}
}

/*
Retorna a configuração. Se o arquivo de configuração existir mas estiver
corrompido, o erro de parse é propagado em vez de devolver uma struct
zerada silenciosamente.
*/
func GetConfig() (Config, error) {
	// Caminho do projeto
	configPath := GetDsyncPath()
	// Caminho das configurações do projeto
	configFile := filepath.Join(configPath, "config.json")

	data, err := os.ReadFile(configFile)
	if err != nil {
		appConfig := GetEmptyConfig()
		SaveConfiguration(appConfig)
		return appConfig, nil
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		slog.Error("Error on load configuration")
		return Config{}, err
	}

	return config, nil
}

func GetDsyncPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	// Sistema operacional
	currentOS := runtime.GOOS
	slog.Info("Running in OS: " + currentOS)

	// Caminho padrão de configurações
	appPath := filepath.Join(home, ".dsync")
	exists, err := filesystem.Exists(appPath)
	if err != nil {
		panic(err)
	}
	if !exists {
		os.MkdirAll(appPath, 0755)
	}

	// Caminho das configurações
	return appPath
}
