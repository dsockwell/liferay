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
	maxRH  	= float32(80.0)

	maxFanSpeed	= uint8(90)
	minFanSpeed	= uint8(65)

	ledIncrement 	= uint8(1)
	fanIncrement	= uint8(1)
	
}

/*
func PanelInit() {
	// linechart for temperature
	LCTempM := ui.NewLineChart()
	LCTempM.BorderLabel = "Temperature (C) - Minute"
	LCTempM.Mode = "dot"
	LCTempM.Data = make([]float64, 60)

	LCTempH := ui.NewLineChart()
	LCTempH.BorderLabel = "Temperature (C) - Hour"
	LCTempH.Mode = "dot"
	LCTempH.Data = make([]float64, 60)
	
	// linechart for humidity
	LCHumM := ui.NewLineChart()
	LCHumM.BorderLabel = "Percent RH - Minute"
	LCHumM.Mode = "dot"
	LCHumM.Data = make([]float64, 60)

	LCHumH := ui.NewLineChart()
	LCHumH.BorderLabel = "Percent RH - Hour"
	LCHumH.Mode = "dot"
	LCHumH.Data = make([]float64, 60)
	
	// linechart for fanspeed
	LCFanM := ui.NewLineChart()
	LCFanM.BorderLabel = "Percent Fan Speed - Minute"
	LCFanM.Mode = "dot"
	LCFanM.Data = make([]float64, 60)

	LCFanH := ui.NewLineChart()
	LCFanH.BorderLabel = "Percent Fan Speed - Hour"
	LCFanH.Mode = "dot"
	LCFanH.Data = make([]float64, 60)
	
	ui.Body.AddRows(
		ui.NewRow(
			ui.NewCol(2, 0, LCTempH),
			ui.NewCol(2, 0, LCTempM)),
		ui.NewRow(
			ui.NewCol(2, 0, LCHumH),
			ui.NewCol(2, 0, LCHumM)),
		ui.NewRow(
			ui.NewCol(2, 0, LCFanH),
			ui.NewCol(2, 0, LCFanM)))
	ui.Body.Align()
	ui.Render(ui.Body)
}
	
*/

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

                gobot.Every(1*time.Second, func() {
			tempInst	= bme280.Temperature
			humInst		= bme280.Humidity
			
			// function:
			// Maximize light intensity
			// Avoid max temp
			// LifeRay controls temp
			// Avoid min RH
			// maintain minRH < RH < maxRH
			// Fan controls RH

			if !daylight {
				return
			}

			if tempInst > maxTemp {
				Warning.Println("[!] High temperature", tempInst)
				// cool down		
				if LRBrightness > minLRBrightness  {
					LRBrightness = minLRBrightness
					Warning.Println("[*] LifeRay intensity decreased to",
							 LRBrightness)
				}
			} else if LRBrightness > 0 && LRBrightness < maxLRBrightness {
				LRBrightness += ledIncrement
			}
			if LRBrightness < 0 {
				LRBrightness = 0
			}
			if LRBrightness > maxLRBrightness {
				LRBrightness = maxLRBrightness
			}
		})
                gobot.Every(10*time.Second, func() {
			
			fan.Brightness(fanspeed)

			if humInst < minRH && fanspeed > minFanSpeed {
				Warning.Println("[!] Low RH", humInst)
				fanspeed = minFanSpeed
				Warning.Println("[*] Fan speed decreased to", fanspeed)
			} else if humInst > minRH {
				fanspeed += fanIncrement
			}
				

			if fanspeed < minFanSpeed {
				fanspeed = minFanSpeed
			}
			if fanspeed > maxFanSpeed {
				fanspeed = maxFanSpeed
			}

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
