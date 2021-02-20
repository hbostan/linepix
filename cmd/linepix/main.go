package main

import (
	"fmt"
	"image"
	"io"
	"log"
	"math"
	"math/rand"
	"os"
	"os/exec"
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
	rand.Seed(time.Now().Unix())

	var input, output string
	var lineCount, precision, lineWeight, fps, freeze int
	var width, height uint
	var color, video bool
	// Defaults
	lineCount = 2500
	precision = 60
	lineWeight = 16
	width = 512
	height = 0
	color = false
	video = false
	fps = 3000
	freeze = 5

	helpFlag := getopt.BoolLong("help", 0, "Display help.")
	getopt.FlagLong(&input, "input", 'i', "Name of the input image").Mandatory()
	getopt.FlagLong(&output, "output", 'o', "Name of the output image/video").Mandatory()
	getopt.FlagLong(&lineCount, "line-count", 'l', "Number of lines to draw.")
	getopt.FlagLong(&precision, "precision", 'p', "Number of tests performed to find the darkest line.")
	getopt.FlagLong(&width, "width", 'w', "Width of the output image.")
	getopt.FlagLong(&height, "height", 'h', "Height of the output image.")
	getopt.FlagLong(&color, "color", 'c', "Generate color image.")
	getopt.FlagLong(&fps, "fps", 'f', "FPS value for the generated video. Each frame adds a single line.")
	getopt.FlagLong(&freeze, "freeze", 's', "Time to freeze the last frame of the video.")
	getopt.FlagLong(&video, "video", 'v', "Generate video of the process.")
	getopt.ParseV2()

	if *helpFlag {
		getopt.PrintUsage(os.Stdout)
		os.Exit(0)
	}
	if strings.HasSuffix(output, ".png") {
		output = output[:len(output)-4]
	}

	linech := make(chan linepix.Line, lineCount)

	inpImage := linepix.ReadImage(input)
	// Don't resize if width & height are both 0
	if width != 0 || height != 0 {
		inpImage = linepix.ResizeImage(inpImage, width, height)
		log.Printf("📏 Image Resized (%vx%v)\n", inpImage.Bounds().Dx(), inpImage.Bounds().Dy())
	}
	var ffmpeg *exec.Cmd
	var ffmpegStdin io.WriteCloser
	if video {
		var err error
		ffmpeg, ffmpegStdin, err = linepix.SetupFFMPEG(output+".mp4", inpImage.Bounds().Dx(), inpImage.Bounds().Dy(), fps, freeze)
		if err != nil {
			log.Println("Failed running FFMPEG. Not generating video.")
			video = false
		}
	}

	if color {
		log.Println("🌈 Processing color image.")
		rgba := linepix.MakeRGBA(inpImage)
		log.Println("🌀  Converted to RGBA")
		red, green, blue := linepix.SplitImage(rgba)
		log.Println("🪓 Splitted color channels")
		redLum := linepix.TotalLuminosity(red)
		greenLum := linepix.TotalLuminosity(green)
		blueLum := linepix.TotalLuminosity(blue)
		rr, gr, br := linepix.CalculateChannelRatios(redLum, greenLum, blueLum, len(red.Pix))
		rLineCount := int(float64(lineCount) * rr)
		gLineCount := int(float64(lineCount) * gr)
		bLineCount := int(float64(lineCount) * br)
		lineCount = rLineCount + gLineCount + bLineCount
		log.Println("🧮 Calculated number of lines to draw for each channel")
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
	if video && ffmpeg != nil {
		err := ffmpeg.Start()
		if err != nil {
			log.Println("Cannot start FFMPEG. Not generating video.")
			video = false
		} else {
			log.Println("🎥 Started ffmpeg.")
		}
	}

	log.Println("🧙‍♂️ Working my magic!✨")

	start := time.Now()
	for i := 0; i < lineCount; i++ {
		var line linepix.Line
		line = <-linech
		linepix.DrawLine(outImage, line, true)
		if video && ffmpeg != nil {
			ffmpegStdin.Write(outImage.Pix)
		}
		percent := float64(i) / float64(lineCount)
		elapsed := time.Since(start)
		estimated := start.Add(time.Duration(float64(elapsed) / percent))
		eta := estimated.Sub(time.Now())
		l := fmt.Sprintf("%6v/%6v Gen Lines %15v", i, lineCount, eta.Truncate(time.Second))
		coloredBackground(l, percent)
	}
	if video && ffmpeg != nil {
		ffmpegStdin.Close()
	}
	fmt.Println()
	linepix.SaveImage(output+".png", outImage)
	fmt.Printf("✅ Done! ✅ (Took %v)\n", time.Since(start).Truncate(time.Millisecond))
}
