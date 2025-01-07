package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// visit 函数用于处理每个访问到的文件或目录
func visit(path string, info os.FileInfo, err error) error {
	if err != nil {
		fmt.Println(err) // 打印错误信息
		return err
	}
	if !info.IsDir() && filepath.Ext(path) == ".md" && !strings.Contains(path, "README") {
		fmt.Println("找到文件:", info.Name(), " 开始处理")
		newFunction(path, info.Name())
	}
	return nil
}

var dir string
var pre string

func main() {
	// 指定Markdown文件路径

	flag.StringVar(&dir, "dir", "./", "文件夹")
	flag.StringVar(&pre, "pre", "https://raw.gitcode.com/mouuii/k8s-learning/raw/main", "替换的连接")
	flag.Parse()
	err := filepath.Walk(dir, visit)
	if err != nil {
		fmt.Println("Error walking the path:", err)
		return
	}
}

func newFunction(markdownFilePath string, fileName string) bool {

	file, err := os.Open(markdownFilePath)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return true
	}
	defer file.Close()

	outputFile, err := os.OpenFile(fileName+"_generate.md", os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error creating file:", err)
		return true
	}
	defer outputFile.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		if strings.Contains(line, `<img src="`) {

			line = strings.Replace(line, `src="../..`, fmt.Sprintf(`src="%s`, pre), 1)
		}

		outputFile.WriteString(line + "\n")
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading file:", err)
	}
	return false
}
