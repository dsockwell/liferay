package main

import (
        "time"
	// "fmt"
	"log"
	"os"
	"io"
	"io/ioutil"
        "github.com/hybridgroup/gobot"
        "github.com/dsockwell/gobot/platforms/beaglebone"
        "github.com/hybridgroup/gobot/platforms/gpio"
        "github.com/dsockwell/gobot/platforms/i2c"

        "github.com/felixge/pidctrl"

	// ui "github.com/gizak/termui"
)

var (
	Trace	*log.Logger
	Info	*log.Logger
	Warning	*log.Logger
	Error	*log.Logger

	LRBrightness		uint8
	minLRBrightness		uint8
	maxLRBrightness		uint8
	fanspeed		uint8
	sunrise			int
	sunset			int
	daylight		bool

	minTemp			float32
	maxTemp			float32
	maxRH			float32
	minRH			float32

	maxFanSpeed 		uint8
	minFanSpeed 		uint8
	ledIncrement		uint8
	fanIncrement		uint8

)

func LogInit(
	traceHandle io.Writer,
	infoHandle io.Writer,
	warningHandle io.Writer,
	errorHandle io.Writer) {

    Trace = log.New(traceHandle,
        "TRACE: ",
        log.Ldate|log.Ltime|log.Lshortfile)

    Info = log.New(infoHandle,
        "INFO: ",
        log.Ldate|log.Ltime|log.Lshortfile)

    Warning = log.New(warningHandle,
        "WARNING: ",
        log.Ldate|log.Ltime|log.Lshortfile)

    Error = log.New(errorHandle,
        "ERROR: ",
        log.Ldate|log.Ltime|log.Lshortfile)
}

func AtmoInit () {
	LRBrightness = 0
	minLRBrightness = 128
	maxLRBrightness = 255

	daylight = false

	fanspeed = 0
	sunrise = 7
	sunset = 23

	maxTemp	= float32(34.0)
	minTemp	= float32(30.0)
	minRH  	= float32(75.0)
	maxRH  	= float32(90.0)

	maxFanSpeed	= uint8(90)
	minFanSpeed	= uint8(65)

	ledIncrement 	= uint8(1)
	fanIncrement	= uint8(1)
}

func main() {

	// LogInit(os.Stdout, os.Stdout, os.Stdout, os.Stderr)
	LogInit(ioutil.Discard, os.Stdout, os.Stdout, os.Stderr)
	// LogInit(ioutil.Discard, ioutil.Discard, ioutil.Discard, ioutil.Discard)

	AtmoInit()

	// err := ui.Init()
	// if err != nil {
		// panic(err)
	// }


	// defer ui.Close()
	
        gbot := gobot.NewGobot()

        beagleboneAdaptor := beaglebone.NewBeagleboneAdaptor("beaglebone")

	bme280 := i2c.NewBME280Driver(beagleboneAdaptor, "bme280", 100*time.Millisecond)

        led := gpio.NewLedDriver(beagleboneAdaptor, "led", "P9_14")
        fan := gpio.NewLedDriver(beagleboneAdaptor, "fan", "P8_13")

	// Controls the Sun and the wind and the weather.
	// 16 hour day, starting at 07.00 local time (America/Denver)
	// and ending at 23.00.

	// jumpstart the fan
	fanspeed = uint8(255)
	
	
        LifeRay := func() {

		myloc, _ := time.LoadLocation("America/Denver")
		now := time.Now()

		tSunrise := time.Date(now.Year(), now.Month(), now.Day(), sunrise, 0, 0, 0, myloc)
		tSunset := time.Date(now.Year(), now.Month(), now.Day(), sunset, 0, 0, 0, myloc)
		Info.Println("[*] Life Ray controller initialized")
		

		tempInst	:= bme280.Temperature
		humInst		:= bme280.Humidity

		LifeRayPIDOut	:= 0.0
		FanPIDOut	:= 0.0

		LifeRayPID := pidctrl.NewPIDController(1.0, 1.0, 1.0)
		LifeRayPID.Set(float64(maxTemp))
		LifeRayPID.SetOutputLimits(0.25, 1.0)

		FanPID := pidctrl.NewPIDController(1.0, 1.0, 1.0)
		FanPID.Set(float64(maxRH))
		FanPID.SetOutputLimits(0.5, 1.0)

                gobot.Every(5*time.Second, func() {
			tempInst	= bme280.Temperature
			
			if !daylight {
				return
			}

			// From here, use PIDs!

			if tempInst > maxTemp {
				Warning.Println("[!] High temperature", tempInst)
			}
			LifeRayPIDOut = LifeRayPID.Update(float64(tempInst))
			// LifeRayPIDOut = LifeRayPID.UpdateDuration(float64(tempInst), 5*time.Second)
			Trace.Println("[=] LifeRay PID output", LifeRayPIDOut)
			LRBrightness = uint8(LifeRayPIDOut * 255)
			
		})

                gobot.Every(5*time.Second, func() {
			humInst		= bme280.Humidity
			

			if humInst < minRH && fanspeed > minFanSpeed {
				Warning.Println("[!] Low RH", humInst)
			}
			FanPIDOut = FanPID.Update(float64(humInst))
			// FanPIDOut = FanPID.UpdateDuration(float64(humInst), 1*time.Second)
			FanPIDOut = 1 - FanPIDOut
			Trace.Println("[=] Fan PID output", FanPIDOut)
			fanspeed = uint8(FanPIDOut * 255)
			fan.Brightness(fanspeed)

		})

		// Lighting loop
                gobot.Every(1*time.Second, func() {
                        led.Brightness(LRBrightness)

			now = time.Now()
                        if now.After(tSunrise) && now.Before(tSunset) {
				Trace.Println("[*] Daylight - Life Ray is active at", now)
				daylight = true
				if LRBrightness == 0 {
					LRBrightness = 255
				}
			} else if now.After(tSunset) && now.After(tSunrise) {
				Trace.Println("[=] Sunset - Life Ray is not active at", now)
				tSunrise = tSunrise.AddDate(0, 0, 1)
				Trace.Println("[=] Advancing calendar. Next Sunrise at", tSunrise)
				tSunset = tSunset.AddDate(0, 0, 1)
				Trace.Println("[=] Advancing calendar. Next Sunset at", tSunset)
				LRBrightness = 0
				daylight = false
			} else {
				Trace.Println("[=] Night Time - Life Ray is not active at", now)
				Trace.Println("[=] Next Sunrise at", tSunrise)
				LRBrightness = 0
				daylight = false
			}
				
                })

		// The sensor loop - am I supposed to be putting these all in the work function?
		gobot.Every(10*time.Second, func() {
			Info.Println("[=] Temperature", bme280.Temperature)
			Info.Println("[=] Humidity", bme280.Humidity)
			Info.Println("[=] LifeRay intensity", LRBrightness)
			Info.Println("[=] Fan speed", fanspeed)
			// Info.Println("[=] Reset sensor:", bme280.Reset())
		})
	}

        LifeRayRobot := gobot.NewRobot("LifeRayBot",
                []gobot.Connection{beagleboneAdaptor},
                []gobot.Device{led},
                []gobot.Device{fan},
                []gobot.Device{bme280},
                LifeRay,
        )

        gbot.AddRobot(LifeRayRobot)

        gbot.Start()
	// PanelInit()
}
