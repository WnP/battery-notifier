// battery-notifier:
//
// This is a simple app which notify your laptop battery state, here are the
// steps with corresponding notification:
//
// - 100% and charging will notify you to unplug
// - 80% and charging will notify you to unplug
// - 20% and discharging will notify to plug
// - 10% and discharging will notify to plug and will hibernate 1min after
//
// This app depends on:
// - [libnotify](https://developer.gnome.org/libnotify/) notify-send
// - [zzz](https://github.com/voidlinux/void-runit/blob/master/zzz) to manage hibernate
package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

const (
	capFile    = "/sys/class/power_supply/BAT0/capacity"
	statusFile = "/sys/class/power_supply/BAT0/status"
	hibernated = iota
	plannedHibernate
	notifyedLow
	good
	notifyedHigh
	notifyedTop
)

var (
	state = hibernated
)

func main() {

	quit := scheduleJob()

	listenSysCall(quit)
}

func listenSysCall(quit chan struct{}) {
	// listen to syscall
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	exitChan := make(chan int)
	go func() {
		for {
			s := <-signalChan
			switch s {
			// kill -SIGHUP XXXX
			case syscall.SIGHUP:
				fmt.Println("hungup")

			// kill -SIGINT XXXX or Ctrl+c
			case syscall.SIGINT:
				fmt.Println("Bye")
				close(quit)
				exitChan <- 0

			// kill -SIGTERM XXXX
			case syscall.SIGTERM:
				fmt.Println("force stop")
				close(quit)
				exitChan <- 0

			// kill -SIGQUIT XXXX
			case syscall.SIGQUIT:
				fmt.Println("stop and core dump")
				close(quit)
				exitChan <- 0

			default:
				fmt.Println("Unknown signal.")
				close(quit)
				exitChan <- 1
			}
		}
	}()

	code := <-exitChan
	os.Exit(code)
}

func scheduleJob() chan struct{} {
	// check Battery state every 5 sec
	ticker := time.NewTicker(1 * time.Minute)
	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				check()
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()
	return quit
}

func check() {
	capacity, isCharging := getInfos()

	switch {
	case capacity == 100:
		if isCharging && state < notifyedTop {
			notify("Please unplug you battery to preserve it", false)
			state = notifyedTop
		} else {
			state = good
		}
	case capacity < 10:
		if isCharging {
			state = good
		} else if state == plannedHibernate {
			state = hibernated
			hibernate()
		} else {
			notify("Battery is under 10%, going to hibernate in 1min", true)
			state = plannedHibernate
		}
	case capacity < 20:
		if isCharging {
			state = good
		} else if state != notifyedLow {
			notify("Battery is under 20%, please plug it", true)
			state = notifyedLow
		}
	case capacity > 80:
		if isCharging && state != notifyedHigh {
			notify("Please unplug you battery to preserve it", false)
			state = notifyedHigh
		} else {
			state = good
		}
	}

	// log.Printf("Battery\tcapacity: %v\tStatus: %s\n", c, s)
}

func notify(body string, critical bool) {
	var icon string
	if critical {
		icon = "/home/scl/Pictures/icons/charge_battery_low.png"
	} else {
		icon = "/home/scl/Pictures/icons/charge_battery_ok.png"
	}
	if err := exec.Command(
		"notify-send", "-i", icon, "Battery", body).Run(); err != nil {

		log.Fatal(err)
	}

}

func hibernate() {
	if err := exec.Command("sudo", "ZZZ").Run(); err != nil {
		log.Fatal(err)
	}
}

func getInfos() (capacity int, isCharging bool) {
	var c []byte
	var s []byte
	var err error

	if c, err = ioutil.ReadFile(capFile); err != nil {
		log.Fatal(err)
	}
	if s, err = ioutil.ReadFile(statusFile); err != nil {
		log.Fatal(err)
	}
	if capacity, err = strconv.Atoi(string(c[:len(c)-1])); err != nil {
		log.Fatal(err)
	}
	if s[0] == 'C' {
		isCharging = true
	} else {
		isCharging = false
	}

	return capacity, isCharging
}
