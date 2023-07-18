package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gin-gonic/gin"
)

func main() {
	endpoint := "http://smartmedibox.skyli.xyz"

	r := gin.Default()

	r.Static("/audio", "./audio")

	// 打开药箱
	r.GET("/medicine-box/open", func(c *gin.Context) {
		token := mqttPublish("$oc/devices/64b5ef75b84c1334befb467a_000000000/sys/messages/up", "open")
		if token.Wait() && token.Error() != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to publish message"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Medicine box opened"})
	})

	// 关闭药箱
	r.GET("/medicine-box/close", func(c *gin.Context) {
		token := mqttPublish("$oc/devices/64b5ef75b84c1334befb467a_000000000/sys/messages/up", "close")
		if token.Wait() && token.Error() != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to publish message"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Medicine box closed"})
	})

	// 播放音频
	r.POST("/medicine-box/audio/play", func(c *gin.Context) {
		file, err := c.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read file"})
			return
		}

		f, err := file.Open()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open file"})
			return
		}
		defer f.Close()

		buf := bytes.NewBuffer(nil)
		if _, err := io.Copy(buf, f); err != nil {
			return
		}

		md5sum := md5.Sum(buf.Bytes())
		md5hex := hex.EncodeToString(md5sum[:])

		// write to disk
		err = ioutil.WriteFile("audio/"+md5hex, buf.Bytes(), 0644)
		if err != nil {
			return
		}

		token := mqttPublish("$oc/devices/64b5ef75b84c1334befb467a_000000000/sys/messages/up", "play "+endpoint+"/"+md5hex)
		if token.Wait() && token.Error() != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to publish message"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Audio played"})
	})

	// 设置闹钟
	r.POST("/medicine-box/alarm/set", func(c *gin.Context) {
		println("alarm set")
		timeStr := c.PostForm("time")
		file, err := c.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read file"})
			log.Fatal(err)
			return
		}

		f, err := file.Open()
		println("file opened", err)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open file"})
			return
		}
		defer f.Close()

		buf := bytes.NewBuffer(nil)
		if _, err := io.Copy(buf, f); err != nil {
			return
		}

		md5sum := md5.Sum(buf.Bytes())
		md5hex := hex.EncodeToString(md5sum[:])

		println(md5hex)

		// write to disk
		err = ioutil.WriteFile("audio/"+md5hex, buf.Bytes(), 0644)
		if err != nil {
			return
		}

		t, err := time.Parse("15:04", timeStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid time format"})
			return
		}
		now := time.Now()
		alarmTime := time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), 0, 0, now.Location())
		if alarmTime.Before(now) {
			alarmTime = alarmTime.AddDate(0, 0, 1)
		}
		duration := alarmTime.Sub(now)
		println(duration)
		go func() {
			time.Sleep(duration)
			token := mqttPublish("$oc/devices/64b5ef75b84c1334befb467a_000000000/sys/messages/up", "play "+endpoint+"/"+md5hex)
			if token.Wait() && token.Error() != nil {
				log.Println("Failed to publish message")
			}
		}()
		c.JSON(http.StatusOK, gin.H{"message": "Alarm set"})
	})
	r.Run(":8080")
}

func mqttPublish(topic string, message string) mqtt.Token {
	opts := mqtt.NewClientOptions()
	opts.AddBroker("tcp://6ac6d52656.st1.iotda-device.cn-north-4.myhuaweicloud.com:1883")
	opts.SetClientID("64b5ef75b84c1334befb467a_000000000_0_0_2023071802")
	opts.SetUsername("64b5ef75b84c1334befb467a_000000000")
	opts.SetPassword("4ebb9a5a1cdf38878084b2397fde2b4c6b0acd1d78578a80979db89041bd0aaf")
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal(token.Error())
	}
	defer client.Disconnect(250)
	token := client.Publish(topic, 0, false, message)
	return token
}
