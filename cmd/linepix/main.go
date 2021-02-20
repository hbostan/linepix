package main

import (
	"fmt"
	"image"
	"log"
	"math"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/hbostann/linepix"
	"github.com/pborman/getopt/v2"
)

func coloredBackground(o string, percent float64) {
	whiteBg := color.New(color.BgGreen, color.FgBlack, color.Bold).PrintfFunc()
	blackBg := color.New(color.BgBlack, color.FgGreen, color.Bold).PrintfFunc()
	lineLen := len(o)
	whiteLen := int(math.Ceil(percent * float64(lineLen)))
	whiteBg("\r%s", o[:whiteLen])
	blackBg("%s", o[whiteLen:])
}

func main() {
	var input, output string
	var lineCount, precision, lineWeight int
	var width, height uint
	var color bool
	// Defaults
	lineCount = 2500
	precision = 60
	lineWeight = 16
	width = 512
	height = 0
	color = false

	helpFlag := getopt.BoolLong("help", 0, "Display help.")
	getopt.FlagLong(&input, "input", 'i', "Name of the input image").Mandatory()
	getopt.FlagLong(&output, "output", 'o', "Name of the output image/video").Mandatory()
	getopt.FlagLong(&lineCount, "line-count", 'l', "Number of lines to draw.")
	getopt.FlagLong(&precision, "precision", 'p', "Number of tests performed to find the darkest line.")
	getopt.FlagLong(&width, "width", 'w', "Width of the output image.")
	getopt.FlagLong(&height, "height", 'h', "Height of the output image.")
	getopt.FlagLong(&color, "color", 'c', "Generate color image")
	getopt.ParseV2()

	if *helpFlag {
		getopt.PrintUsage(os.Stdout)
	}
	if strings.HasSuffix(output, ".png") {
		output = output[:len(output)-4]
	}

	linech := make(chan linepix.Line)

	inpImage := linepix.ReadImage(input)
	// Don't resize if width & height are both 0
	if width != 0 || height != 0 {
		inpImage = linepix.ResizeImage(inpImage, width, height)
		log.Printf("📏 Image Resized (%vx%v)\n", inpImage.Bounds().Dx(), inpImage.Bounds().Dy())
	}

	if color {
		log.Println("🌈 Processing color image.")
		rgba := linepix.MakeRGBA(inpImage)
		fmt.Println("🌀  Converted to RGBA")
		red, green, blue := linepix.SplitImage(rgba)
		fmt.Println("🪓 Splitted color channels")
		redLum := linepix.TotalLuminosity(red)
		greenLum := linepix.TotalLuminosity(green)
		blueLum := linepix.TotalLuminosity(blue)
		rr, gr, br := linepix.CalculateChannelRatios(redLum, greenLum, blueLum, len(red.Pix))
		rLineCount := int(float64(lineCount) * rr)
		gLineCount := int(float64(lineCount) * gr)
		bLineCount := int(float64(lineCount) * br)
		lineCount = rLineCount + gLineCount + bLineCount
		fmt.Println("🧮 Calculated number of lines to draw for each channel")
		go linepix.GenerateLines(red, rLineCount, precision, lineWeight, linepix.RedChannel, linech)
		go linepix.GenerateLines(green, gLineCount, precision, lineWeight, linepix.GreenChannel, linech)
		go linepix.GenerateLines(blue, bLineCount, precision, lineWeight, linepix.BlueChannel, linech)
	} else {
		grayscale := linepix.MakeGrayscale(inpImage)
		go linepix.GenerateLines(grayscale, lineCount, precision, lineWeight, linepix.AllChannels, linech)
	}

	outImage := image.NewRGBA(inpImage.Bounds())
	for i := 0; i < len(outImage.Pix); i++ {
		outImage.Pix[i] = 0xff
	}

	log.Println("🧙‍♂️ Working my magic!✨")

	start := time.Now()
	for i := 0; i < lineCount; i++ {
		var line linepix.Line
		line = <-linech
		linepix.DrawLine(outImage, line, true)

		percent := float64(i) / float64(lineCount)
		elapsed := time.Since(start)
		estimated := start.Add(time.Duration(float64(elapsed) / percent))
		eta := estimated.Sub(time.Now())
		l := fmt.Sprintf("%6v/%6v           %15v", i, lineCount, eta.Truncate(time.Second))
		coloredBackground(l, percent)
	}

	fmt.Println()
	linepix.SaveImage(output+".png", outImage)
	fmt.Printf("✅ Done! ✅ (Took %v)\n", time.Since(start).Truncate(time.Millisecond))
}
