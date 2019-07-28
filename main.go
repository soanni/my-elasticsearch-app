package main

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"io"
	"os"
	"path"
	"strconv"
	"time"
)

var (
	logfile os.File
)

func main() {
	argsWithoutProg := os.Args[1:]
	SetupConfigLoad(argsWithoutProg)
	SetupLogging(&logfile)

	hostName := "dl12.aureacentral.com"

	// fmt.Println(getContainersList(hostName))

	startedAfter := time.Now().Unix() - int64(viper.GetInt("periodDays")*3600*24)

	runningContainers := getContainersList(hostName, startedAfter)

	for _, container := range runningContainers {

		fmt.Printf("ID: %s, Name: %s, Created: %s , Status: %s \n", container.ID, container.Names[0], time.Unix(container.Created, 0).Format("2006-01-02"), container.Status)
	}

}

func SetupLogging(file *os.File) {
	timeFormatString := "2006-01-02"
	timeSalt := time.Now().Format(timeFormatString) + "_" + strconv.FormatInt(int64(time.Now().Unix()), 10)
	outputPath := viper.GetString("logPath")

	logFilename := "idle_containers_" + timeSalt + ".log"
	file, err := os.OpenFile(path.Join(outputPath, logFilename), os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		log.Fatal(err)
	}

	mw := io.MultiWriter(os.Stdout, file)
	log.SetOutput(mw)
	log.SetLevel(log.DebugLevel)
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})
}

func SetupConfigLoad(args []string) {
	if len(args) > 0 {
		viper.SetConfigFile(args[0])
	} else {
		viper.SetConfigName("config")       // name of config file (without extension)
		viper.AddConfigPath("/opt/scripts") // path to look for the config file in
		viper.AddConfigPath(".")
	}

	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		log.Fatalf("*** Fatal error config file: %s ***\n", err)
	}
	log.Infof("*** Configuration file used: %s ***\n", viper.ConfigFileUsed())
}
