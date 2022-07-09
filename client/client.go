package main

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
)

func main() {
	delimiter := []byte("}")

	connection, err := net.Dial("tcp", ":8888")
	if err != nil {
		fmt.Printf("Error trying to conect: %s\n", err.Error())
	}
	defer connection.Close()

	userReader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print(">>")
		inputFromUser, err := userReader.ReadString('\n')
		inputFromUser = strings.TrimRight(inputFromUser, "\n")
		if err != nil {
			fmt.Printf("Error reading input: %s\n", err.Error())
			continue
		}

		args := strings.Split(inputFromUser, " ")

		if args[0] == "subscribe" {
			completeTcpMessage := []byte(strings.Join(args[:], " "))
			completeTcpMessage = append(completeTcpMessage, delimiter...)
			connection.Write(completeTcpMessage)
			for {
				reciveFromServer(connection)
			}

		} else if args[0] == "send" {
			file, err := os.Open("./" + args[1])
			if err != nil {
				fmt.Printf("Error trying to open the file: %s\n", err.Error())
				continue
			}

			fileInfo, err := file.Stat()
			if err != nil {
				fmt.Printf("Error trying to get info from file: %s\n", err.Error())
			}

			fileContent := make([]byte, fileInfo.Size())
			n, err := file.Read(fileContent)
			if err != nil {
				fmt.Printf("Error trying to read from file: %s\n", err.Error())
			}

			completeTcpMessage := []byte(strings.Join(args[:], " "))
			completeTcpMessage = append(completeTcpMessage, []byte(" ")...)
			completeTcpMessage = append(completeTcpMessage, []byte(strconv.Itoa(n))...)
			completeTcpMessage = append(completeTcpMessage, delimiter...)
			completeTcpMessage = append(completeTcpMessage, fileContent...)
			connection.Write(completeTcpMessage)

		} else if args[0] == "recive" {
			args[0] = "subscribe"
			completeTcpMessage := []byte(strings.Join(args[:], " "))
			completeTcpMessage = append(completeTcpMessage, delimiter...)
			connection.Write(completeTcpMessage)

			reciveFromServer(connection)
			closeConnection(connection)
			break

		} else if args[0] == "quit" {
			closeConnection(connection)
			break
		} else {
			fmt.Println("Unknoun command...")
		}
	}

}

func reciveFromServer(c net.Conn) {
	myReader := bufio.NewReader(c)
	inputFromServer, err := myReader.ReadBytes(byte(125))

	if err != nil {
		fmt.Printf("Error creaating the New Reader: %s\n", err.Error())
		panic(err)
	}

	command := strings.Split(string(inputFromServer), " ")

	if command[0] == "send" {
		contentLength, err := strconv.Atoi(strings.TrimRight(command[3], "}"))
		if err != nil {
			fmt.Printf("Error obtaining content length: %s\n", err.Error())
		}

		content := make([]byte, contentLength)

		j, err := io.ReadFull(myReader, content)
		if err != nil {
			log.Printf("Error reading the  buffer: %s, bytes: %d", err.Error(), j)
			return
		}

		newFileName := c.LocalAddr().String() + "_" + command[1]
		error := ioutil.WriteFile(newFileName, content[:], 0644)

		if error != nil {
			fmt.Printf("Error writing the incoming file: %s\n", error.Error())
		}
		fmt.Println("File received...")
	} else {
		fmt.Println(strings.TrimRight(string(inputFromServer), "}"))
	}

}

func closeConnection(c net.Conn) {
	fmt.Println("Closing connection")
	c.Close()
}
