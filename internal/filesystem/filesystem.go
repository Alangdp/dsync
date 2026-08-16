package filesystem

import (
	"errors"
	"io/fs"
	"os"
)

/*
Verifica se um caminho existe. O segundo valor de retorno carrega
qualquer erro de I/O que não seja "caminho não existe" (ex.: permissão
negada), pra não ser confundido com "não existe".
*/
func Exists(path string) (bool, error) {
	_, err := os.Stat(path)

	if err == nil {
		return true, nil
	}

	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}

	return false, err
}
