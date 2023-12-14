package utils

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"time"
)

type TRcloneRemote struct {
	remote      string
	taskStarted time.Time
	bytesTotal  int64
	bytesDone   int64
	FName       string
	remotePath  string
	Completed   bool
	Cancel      chan bool
}

func RcloneRemote(remote string) *TRcloneRemote {
	return &TRcloneRemote{
		remote: remote,
	}
}

func (r *TRcloneRemote) DownloadFile(remotePath, destFolder string) {
	cmd := exec.Command("rclone", "cat", r.remote+remotePath) // remote must like "remoteName:"

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Println("Rclone: Error Creating Stranded Out Pipe For Rclone:", err)
		return
	}

	// starting command
	if err := cmd.Start(); err != nil {
		log.Println("Rclone: Error Starting the Command:", err)
		return
	}

	// Create Output File
	r.FName = filepath.Base(remotePath)
	file, err := os.Create(path.Join(destFolder, r.FName))
	if err != nil {
		log.Println("Rclone: Error Creating the Output File", err)
		return
	}

	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	buffer := make([]byte, 1024)
	r.taskStarted = time.Now()
	// Todo : defer delete(TRTasks["downloads"], r.remotePath)
	// TOdo TRTasks["downloads"][Downloads] = r
	r.bytesTotal = int64(r.GetFileSize(remotePath))
	for {
		select {
		case <-r.Cancel:
			log.Println("Rclone: Download Cancelled", r.FName)
			r.Completed = true
			return
		default:
			n, err := stdout.Read(buffer)
			if err != nil && err != io.EOF {
				log.Println("Rclone: Error Reading From Standard Out ", err)
				return
			}

			_, err = file.Write(buffer[:n])
			r.bytesDone += int64(n)

			if n == 0 {
				break
			}
			if err != nil {
				log.Println("Rclone: Error Writing To the Output File:", err)
				break
			}

		}

	}
}

func (r *TRcloneRemote) UploadFile(remotePath, localPath string) {
	cmd := exec.Command("rclone", "copy", localPath, r.remote+remotePath, "--progress")
	stat, err := os.Stat(localPath)
	if err != nil {
		log.Println("Rclone: Cant Open Local File")
		return
	}
	r.bytesTotal = stat.Size()

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Println("Rclone: Error Creating Stranded Out Pipe For Rclone:", err)
		return
	}

	// starting command
	err = cmd.Start()
	if err != nil {
		log.Println("Rclone: Error Starting the Command:", err)
		return
	}
	buffer := make([]byte, 100)
	re := regexp.MustCompile(`(\d+)%`)
	defer func() { r.Completed = true }()
	for {
		select {
		case <-r.Cancel:
			log.Println("Rclone: Download Cancelled", r.FName)
			r.Completed = true
			_ = cmd.Process.Kill()
			return
		default:
			n, err := stdout.Read(buffer)
			if err != nil && err != io.EOF {
				log.Println("Rclone: Error Reading From Standard Out ", err)
				return
			}

			log.Println(n, string(buffer[:n]))

			matches := re.FindStringSubmatch(string(buffer[:n]))
			if processState := cmd.ProcessState; processState != nil {
				if processState.Exited() {
					log.Println("Rclone: Process has exited")
					return
				}
			}

			if n == 0 {
				return
			}

			if len(matches) >= 2 {
				// Extract the percentage value from the matched string
				percentageStr := matches[1]

				// Convert the percentage string to an integer
				percentage, err := strconv.Atoi(percentageStr)
				if err != nil {
					continue
				}

				if percentage != 0 {
					r.bytesDone = (r.bytesTotal * int64(percentage)) / 100
				}
			}
		}
	}

}

func (r *TRcloneRemote) GetFileSize(remotePath string) int {
	cmd := exec.Command("rclone", "size", r.remote+remotePath)
	output, err := cmd.Output()
	if err != nil {
		return 0
	}

	// Regular expression to find the byte size
	regex := regexp.MustCompile(`Total size: .* \((\d+) Byte\)`)

	// FindStringSubmatch returns an array containing the whole match and the submatches
	matches := regex.FindStringSubmatch(string(output))
	if len(matches) >= 2 {
		byteSizeStr := matches[1] // Extracting the byte size as a string
		byteSize, err := strconv.Atoi(byteSizeStr)
		if err != nil {
			fmt.Println("Rclone : Error converting string to int:", err)
			return 0
		}
		return byteSize
	}
	return 0
}

func (r *TRcloneRemote) GetJson(remotePath string) {

}

func GenerateRcloneString(rTask *TRcloneRemote) string {
	result := fmt.Sprintf("Bytes Total: %d\nBytes Done: %d\nFile Name: %s\n",
		rTask.bytesTotal,
		rTask.bytesDone,
		rTask.FName)
	return result
}
