package main

import (
	"fmt"
	"image"
	"log"
	"os"
	"strings"
	"time"

	"github.com/hbostann/linepix"
	"github.com/pborman/getopt/v2"
)

func main() {
	var input, output string
	var lineCount, precision, lineWeight int
	var width, height uint
	// Defaults
	lineCount = 2500
	precision = 60
	lineWeight = 16
	width = 512
	height = 0

	helpFlag := getopt.BoolLong("help", 0, "Display help.")
	getopt.FlagLong(&input, "input", 'i', "Name of the input image").Mandatory()
	getopt.FlagLong(&output, "output", 'o', "Name of the output image/video").Mandatory()
	getopt.FlagLong(&lineCount, "line-count", 'l', "Number of lines to draw.")
	getopt.FlagLong(&precision, "precision", 'p', "Number of tests performed to find the darkest line.")
	getopt.FlagLong(&width, "width", 'w', "Width of the output image.")
	getopt.FlagLong(&height, "height", 'h', "Height of the output image.")
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
		log.Println("📏 Image Resized (%vx%v)\n", inpImage.Bounds().Dx(), inpImage.Bounds().Dy())
	}
	grayscale := linepix.MakeGrayscale(inpImage)
	go linepix.GenerateLines(grayscale, lineCount, precision, lineWeight, linech)

	outImage := image.NewGray(inpImage.Bounds())
	for i := 0; i < len(outImage.Pix); i++ {
		outImage.Pix[i] = 0xff
	}

	log.Println("🧙‍♂️ Working my magic!✨")

	start := time.Now()
	for i := 0; i < lineCount; i++ {
		var line linepix.Line
		line = <-linech
		linepix.DrawLine(outImage, line)
	}

	fmt.Println()
	linepix.SaveImage(output+".png", outImage)
	fmt.Printf("✅ Done! ✅ (Took %v)\n", time.Since(start).Truncate(time.Millisecond))
}
