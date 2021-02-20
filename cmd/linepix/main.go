package main

import (
	"os"

	"github.com/hbostann/linepix"
	"github.com/pborman/getopt/v2"
)

func main() {
	var input, output string

	helpFlag := getopt.BoolLong("help", 0, "Display help.")
	getopt.FlagLong(&input, "input", 'i', "Name of the input image").Mandatory()
	getopt.FlagLong(&output, "output", 'o', "Name of the output image/video").Mandatory()
	getopt.ParseV2()
	if *helpFlag {
		getopt.PrintUsage(os.Stdout)
	}

	image := linepix.ReadImage(input)
	image = linepix.ResizeImage(image, 0, 250)
	linepix.SaveImage(output, image)

}
