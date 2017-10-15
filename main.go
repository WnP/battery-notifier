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
	capacity   = "/sys/class/power_supply/BAT0/capacity"
	status     = "/sys/class/power_supply/BAT0/status"
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

	// listen to syscall
	signal_chan := make(chan os.Signal, 1)
	signal.Notify(signal_chan,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	exit_chan := make(chan int)
	go func() {
		for {
			s := <-signal_chan
			switch s {
			// kill -SIGHUP XXXX
			case syscall.SIGHUP:
				fmt.Println("hungup")

			// kill -SIGINT XXXX or Ctrl+c
			case syscall.SIGINT:
				fmt.Println("Bye")
				close(quit)
				exit_chan <- 0

			// kill -SIGTERM XXXX
			case syscall.SIGTERM:
				fmt.Println("force stop")
				close(quit)
				exit_chan <- 0

			// kill -SIGQUIT XXXX
			case syscall.SIGQUIT:
				fmt.Println("stop and core dump")
				close(quit)
				exit_chan <- 0

			default:
				fmt.Println("Unknown signal.")
				close(quit)
				exit_chan <- 1
			}
		}
	}()

	code := <-exit_chan
	os.Exit(code)
}

func check() {
	c, s := getInfos()

	switch {
	case c == 100:
		switch s {
		case "Charging":
			if state < notifyedTop {
				notify("Please unplug you battery to preserve it", false)
				state = notifyedTop
			}
		case "Discharging":
			state = good
		}
	case c < 10:
		switch s {
		case "Charging":
			state = good
		case "Discharging":
			if state == plannedHibernate {
				hibernate()
			} else {
				notify(
					"Battery is under 10%, going to hibernate in 1min", true)
				state = plannedHibernate
			}
		}
	case c < 20:
		switch s {
		case "Charging":
			state = good
		case "Discharging":
			notify("Battery is under 20%, please plug it", true)
			state = notifyedLow
		}
	case c > 80:
		switch s {
		case "Charging":
			if state < notifyedHigh {
				notify("Please unplug you battery to preserve it", false)
				state = notifyedHigh
			}
		case "Discharging":
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

func getInfos() (c int, s string) {
	var cap []byte
	var stat []byte
	var err error

	if cap, err = ioutil.ReadFile(capacity); err != nil {
		log.Fatal(err)
	}
	if stat, err = ioutil.ReadFile(status); err != nil {
		log.Fatal(err)
	}
	if c, err = strconv.Atoi(string(cap[:len(cap)-1])); err != nil {
		log.Fatal(err)
	}

	stat = stat[:len(stat)-1]

	return c, string(stat)
}
