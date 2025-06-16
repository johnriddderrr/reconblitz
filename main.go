package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

func checkTool(tool string) error {
	_, err := exec.LookPath(tool)
	return err
}

func runCommand(name, cmd string, args []string, outputDir string, wg *sync.WaitGroup, results chan<- string) {
	defer wg.Done()
	fmt.Printf("[*] Запускаем %s...\n", name)

	outFile := filepath.Join(outputDir, name+"_detailed.log")
	f, err := os.Create(outFile)
	if err != nil {
		results <- fmt.Sprintf("%s: ошибка записи лога: %v", name, err)
		return
	}
	defer f.Close()

	command := exec.Command(cmd, args...)
	command.Stdout = f
	command.Stderr = f

	err = command.Run()
	if err != nil {
		results <- fmt.Sprintf("%s: ошибка выполнения: %v", name, err)
		return
	}
	results <- fmt.Sprintf("%s: выполнено успешно", name)
}

func main() {
	tools := map[string][]string{
		"subfinder": {"-d"},
		"rustscan":  {"-a", "--", "-sV", "-sC"},
		"nuclei":    {"-u"},
	}

	// Проверяем инструменты
	for tool := range tools {
		if err := checkTool(tool); err != nil {
			fmt.Printf("Ошибка: инструмент '%s' не найден в PATH. Установи его и попробуй снова.\n", tool)
			return
		}
	}

	var target string
	if len(os.Args) > 1 {
		target = os.Args[1]
	} else {
		fmt.Print("Введите домен или IP для сканирования: ")
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Ошибка чтения ввода:", err)
			return
		}
		target = input[:len(input)-1] // удаляем \n
	}

	timestamp := time.Now().Format("20060102_150405")
	outputDir := "results_" + target + "_" + timestamp
	err := os.Mkdir(outputDir, 0755)
	if err != nil {
		fmt.Println("Ошибка создания папки для результатов:", err)
		return
	}

	var wg sync.WaitGroup
	resultsChan := make(chan string, len(tools))

	for name, args := range tools {
		wg.Add(1)
		// Формируем аргументы с добавлением цели
		argsFull := append(args, target)
		go runCommand(name, name, argsFull, outputDir, &wg, resultsChan)
	}

	wg.Wait()
	close(resultsChan)

	// Пишем краткий лог summary.txt
	summaryFile := filepath.Join(outputDir, "summary.txt")
	fSummary, err := os.Create(summaryFile)
	if err != nil {
		fmt.Println("Ошибка создания summary.txt:", err)
		return
	}
	defer fSummary.Close()

	for res := range resultsChan {
		fmt.Println(res)
		fSummary.WriteString(res + "\n")
	}

	fmt.Println("[+] Сканирование завершено. Результаты в папке:", outputDir)
}
