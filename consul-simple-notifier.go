package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/pelletier/go-toml"
	//"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
)

type config struct {
	emails     []string
	ikachanUrl string
	channel    string
}

type consulAlert struct {
	Timestamp string
	Node      string
	ServiceId string
	Service   string
	CheckId   string
	Check     string
	Output    string
	Notes     string
}

const (
	version = "0.0.1"
)

var (
	logger = log.New(os.Stdout, "[ikachan-proxy] ", log.LstdFlags)
)

func main() {
	var (
		justShowVersion bool
		configPath      string
		conf            config
		input           []consulAlert
	)

	flag.BoolVar(&justShowVersion, "v", false, "Show version")
	flag.BoolVar(&justShowVersion, "version", false, "Show version")

	flag.StringVar(&configPath, "c", "/etc/consul-simple-notifier.ini", "Config path")
	flag.Parse()

	if justShowVersion {
		showVersion()
		return
	}

	parsed, err := toml.LoadFile(configPath)
	if err != nil {
		panic(err.Error())
	}
	recipients := parsed.Get("email.recipients")
	for _, address := range recipients.([]interface{}) {
		conf.emails = append(conf.emails, address.(string))
	}
	conf.ikachanUrl = parsed.Get("ikachan.url").(string)
	conf.channel = parsed.Get("ikachan.channel").(string)
	logger.Printf("%+v\n", conf)

	err = json.NewDecoder(os.Stdin).Decode(&input)
	if err != nil {
		panic(err.Error())
	}
	logger.Printf("%+v\n", input)

	for _, content := range input {
		notifyEmail(conf.emails, content)
		notifyIkachan(conf.ikachanUrl, conf.channel, content)
	}
}

func notifyEmail(recipients []string, content consulAlert) error {
	for _, address := range recipients {
		logger.Printf("Sending... %s to %+v\n", address, content)
		cmd := exec.Command("/bin/mail", "-s", "Alert from consul", address)
		stdin, err := cmd.StdinPipe()
		if err != nil {
			return err
		}

		if err := cmd.Start(); err != nil {
			return err
		}

		fmt.Fprintf(stdin, "This is a sample mail\n%+v", content)
		stdin.Close()
		logger.Printf("Send!\n")
		cmd.Wait()
	}
	return nil
}

func notifyIkachan(ikachanUrl string, channel string, content consulAlert) error {
	joinUrl := fmt.Sprintf("%s/join", ikachanUrl)
	noticeUrl := fmt.Sprintf("%s/notice", ikachanUrl)

	values := make(url.Values)
	values.Set("channel", channel)

	resp1, err := http.PostForm(joinUrl, values)
	defer resp1.Body.Close()
	if err != nil {
		return err
	}

	message := fmt.Sprintf("This is a sample notification! %s - %s", content.CheckId, content.Output)
	values.Set("message", message)

	logger.Printf("Posted! %+v", values)
	resp2, err := http.PostForm(noticeUrl, values)
	defer resp2.Body.Close()
	if err != nil {
		return err
	}

	return nil
}

func showVersion() {
	fmt.Printf("consul-simple-notifier version: %s\n", version)
}
